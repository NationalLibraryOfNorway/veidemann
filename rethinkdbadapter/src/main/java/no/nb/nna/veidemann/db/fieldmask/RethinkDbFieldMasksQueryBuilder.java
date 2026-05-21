/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.db.fieldmask;

import com.google.protobuf.Descriptors.FieldDescriptor.Type;
import com.google.protobuf.MessageOrBuilder;
import com.google.protobuf.Timestamp;
import com.rethinkdb.gen.ast.ReqlExpr;
import com.rethinkdb.gen.ast.ReqlFunction1;
import com.rethinkdb.gen.ast.Table;
import com.rethinkdb.model.MapObject;
import no.nb.nna.veidemann.db.ProtoUtils;
import no.nb.nna.veidemann.db.fieldmask.Indexes.Index;
import no.nb.nna.veidemann.db.fieldmask.MaskedObject.UpdateType;
import no.nb.nna.veidemann.db.queryoptimizer.QueryOptimizer;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import static com.rethinkdb.RethinkDB.r;

public abstract class RethinkDbFieldMasksQueryBuilder<T extends MessageOrBuilder> {
    private final Indexes<T> indexes = new Indexes<>();
    private final List<String> readOnlyPaths = new ArrayList<>();
    private final List<String> minimumReturnedFields = new ArrayList<>();
    private final ObjectOrMask<T> maskedObject;

    public RethinkDbFieldMasksQueryBuilder(ObjectOrMask<T> maskedObject) {
        this.maskedObject = maskedObject;
        init();
    }

    protected abstract void init();

    protected void addIndex(String indexName, String... path) {
        indexes.addIndex(indexName, path);
    }

    protected void addIgnoreCaseIndex(String indexName, String... path) {
        indexes.addIgnoreCaseIndex(indexName, path);
    }

    protected void addPrimaryIndex(String indexName, String path) {
        indexes.addPrimaryIndex(indexName, path);
    }

    public Index getPrimaryIndex() {
        return indexes.getPrimary();
    }

    public ObjectOrMask<T> getMaskedObject() {
        return maskedObject;
    }

    public Index getBestIndex() {
        return indexes.getBestIndex(maskedObject);
    }

    public List<Index> getBestIndexes() {
        return indexes.getBestIndexes(maskedObject);
    }

    public List<Index> getBestIndexes(String... path) {
        return indexes.getBestIndexes(path);
    }

    protected void addReadOnlyPath(String path) {
        readOnlyPaths.add(path);
    }

    protected void addMinimumReturnedField(String path) {
        minimumReturnedFields.add(path);
    }

    public ReqlExpr createOrderByQuery(ReqlExpr q, String fieldName, String indexName, boolean descending) {
        if (q instanceof Table && indexName != null) {
            if (descending) {
                q = q.orderBy().optArg("index",
                        r.desc(indexName));
            } else {
                q = q.orderBy().optArg("index",
                        r.asc(indexName));
            }
        } else {
            if (descending) {
                q = q.orderBy(r.desc(fieldName));
            } else {
                q = q.orderBy(r.asc(fieldName));
            }
        }
        return q;
    }

    public List<Object> createPluckQuery() {
        List<Object> pluckQuery = new ArrayList<>(minimumReturnedFields);
        maskedObject.getMasks().children.forEach(e -> innerCreatePluckQuery(pluckQuery, e));
        return pluckQuery;
    }

    private void innerCreatePluckQuery(List<Object> pluckQuery, PathElem<T> pathElem) {
        if (maskedObject.getPathDef(pathElem.fullName) != null) {
            pluckQuery.add(pathElem.name);
        } else {
            List<Object> childQuery = r.array();
            pluckQuery.add(r.hashMap(pathElem.name, childQuery));
            pathElem.children.forEach(c -> innerCreatePluckQuery(childQuery, c));
        }
    }

    public void elems(QueryOptimizer<T> optimizer, T queryTemplate) {
        for (PathElem<T> pathElem : maskedObject.getPaths()) {
            Object value = pathElem.getValue(queryTemplate);
            List<Object> values = r.array();
            if (pathElem.descriptor.isRepeated()) {
                for (Object entry : (List<?>) ProtoUtils.protoFieldToRethink(pathElem.descriptor, value)) {
                    values.add(entry);
                }
            } else {
                values.add(ProtoUtils.protoFieldToRethink(pathElem.descriptor, value));
            }
            optimizer.wantMaskElem(pathElem.fullName, values);
        }
    }

    public ReqlFunction1 buildFilterQuery(T queryTemplate) {
        return row -> {
            ReqlExpr expression = row;
            boolean first = true;
            for (PathElem<T> pathElem : maskedObject.getPaths()) {
                if (first) {
                    expression = innerBuildFilterQuery(row, pathElem, queryTemplate);
                } else {
                    expression = expression.and(innerBuildFilterQuery(row, pathElem, queryTemplate));
                }
                first = false;
            }
            return expression;
        };
    }

    private ReqlExpr innerBuildFilterQuery(ReqlExpr expression, PathElem<T> pathElem, T queryTemplate) {
        expression = buildGetFieldExpression(pathElem, expression);
        Object value = pathElem.getValue(queryTemplate);
        if (pathElem.descriptor.isRepeated()) {
            List<Object> values = r.array();
            for (Object entry : (List<?>) ProtoUtils.protoFieldToRethink(pathElem.descriptor, value)) {
                values.add(entry);
            }
            expression = expression.contains(r.args(values));
        } else {
            expression = expression.eq(ProtoUtils.protoFieldToRethink(pathElem.descriptor, value));
        }
        return expression;
    }

    public ReqlExpr buildGetFieldExpression(PathElem<T> pathElem, ReqlExpr parentExpr) {
        if (!pathElem.parent.name.isEmpty()) {
            parentExpr = buildGetFieldExpression(pathElem.parent, parentExpr);
        }
        return parentExpr.g(pathElem.name);
    }

    public ReqlFunction1 buildUpdateQuery(T object) {
        return row -> {
            MapObject<Object, Object> updateQuery = r.hashMap();
            for (PathElem<T> pathElem : maskedObject.getMasks().children) {
                innerBuildUpdateQuery(row, updateQuery, pathElem, object);
            }
            return updateQuery;
        };
    }

    private void innerBuildUpdateQuery(ReqlExpr row, Map<Object, Object> updateQuery, PathElem<T> pathElem, T object) {
        if (readOnlyPaths.contains(pathElem.fullName)) {
            return;
        }

        if (pathElem.descriptor.isRepeated()) {
            PathElem<T> pathDef = maskedObject.getPathDef(pathElem.fullName);
            if (pathDef == null || pathDef.updateType == UpdateType.REPLACE) {
                updateQuery.put(pathElem.name, ProtoUtils.protoFieldToRethink(pathElem.descriptor, pathElem.getValue(object)));
            } else if (pathDef.updateType == UpdateType.APPEND) {
                updateQuery.put(pathElem.name, buildGetFieldExpression(pathDef, row).default_(r.array())
                        .setUnion(ProtoUtils.protoFieldToRethink(pathElem.descriptor, pathElem.getValue(object))));
            } else {
                updateQuery.put(pathElem.name, buildGetFieldExpression(pathDef, row).default_(r.array())
                        .setDifference(ProtoUtils.protoFieldToRethink(pathElem.descriptor, pathElem.getValue(object))));
            }
        } else if (pathElem.descriptor.getType() == Type.MESSAGE) {
            if (pathElem.descriptor.isRepeated()
                    || pathElem.parent.name.isEmpty()
                    || ((MessageOrBuilder) pathElem.parent.getValue(object)).hasField(pathElem.descriptor)) {
                if (pathElem.descriptor.getMessageType() == Timestamp.getDescriptor()) {
                    updateQuery.put(pathElem.name, ProtoUtils.protoFieldToRethink(pathElem.descriptor, pathElem.getValue(object)));
                } else {
                    MapObject<Object, Object> childQuery = r.hashMap();
                    updateQuery.put(pathElem.name, childQuery);
                    pathElem.children.forEach(c -> innerBuildUpdateQuery(row, childQuery, c, object));
                }
            } else {
                updateQuery.put(pathElem.name, null);
            }
        } else {
            updateQuery.put(pathElem.name, ProtoUtils.protoFieldToRethink(pathElem.descriptor, pathElem.getValue(object)));
        }
    }
}
