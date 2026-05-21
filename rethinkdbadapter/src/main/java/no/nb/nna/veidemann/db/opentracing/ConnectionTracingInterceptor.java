/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.db.opentracing;

import com.fasterxml.jackson.core.type.TypeReference;
import com.rethinkdb.ast.ReqlAst;
import com.rethinkdb.gen.ast.Datum;
import com.rethinkdb.model.OptArgs;
import com.rethinkdb.net.Connection;
import com.rethinkdb.net.Result;
import io.opentracing.Scope;
import io.opentracing.Span;
import io.opentracing.Tracer;
import io.opentracing.noop.NoopSpan;
import io.opentracing.tag.Tags;
import io.opentracing.util.GlobalTracer;

/**
 *
 */
public class ConnectionTracingInterceptor extends Connection {

    public static final String OPERATION_NAME_KEY = "opertaionName";

    private final Connection conn;

    private final boolean withActiveSpanOnly;

    public ConnectionTracingInterceptor(Connection conn) {
        this(conn, false);
    }

    public ConnectionTracingInterceptor(Connection conn, boolean withActiveSpanOnly) {
        super(new Connection.Builder());
        this.conn = conn;
        this.withActiveSpanOnly = withActiveSpanOnly;
    }

    @Override
    public void runNoReply(ReqlAst term, OptArgs globalOpts) {
        Tracer tracer = getTracer();
        Span span = buildSpan(tracer, globalOpts, "runNoReply");
        try (Scope scope = tracer.scopeManager().activate(span)) {
            conn.runNoReply(term, globalOpts);
        } finally {
            span.finish();
        }
    }

    @Override
    public <T> Result<T> run(ReqlAst term, OptArgs globalOpts, Result.FetchMode fetchMode, Boolean unwrap,
                             TypeReference<T> typeRef) {
        Tracer tracer = getTracer();
        Span span = buildSpan(tracer, globalOpts, "run");
        try (Scope scope = tracer.scopeManager().activate(span)) {
            return conn.run(term, globalOpts, fetchMode, unwrap, typeRef);
        } finally {
            span.finish();
        }
    }

    @Override
    public void noreplyWait() {
        conn.noreplyWait();
    }

    @Override
    public Connection use(String db) {
        conn.use(db);
        return this;
    }

    @Override
    public void close(boolean shouldNoreplyWait) {
        conn.close(shouldNoreplyWait);
    }

    @Override
    public void close() {
        conn.close();
    }

    @Override
    public boolean isOpen() {
        return conn.isOpen();
    }

    @Override
    public Connection reconnect(boolean noreplyWait) {
        conn.reconnect(noreplyWait);
        return this;
    }

    @Override
    public Connection reconnect() {
        conn.reconnect();
        return this;
    }

    @Override
    public Connection connect() {
        conn.connect();
        return this;
    }

    @Override
    public String db() {
        return conn.db();
    }

    Tracer getTracer() {
        return GlobalTracer.get();
    }

    Span buildSpan(Tracer tracer, OptArgs globalOpts, String defaultOperationName) {
        String operationName;
        ReqlAst operationNameDatum = globalOpts.remove(OPERATION_NAME_KEY);

        if (withActiveSpanOnly && tracer.scopeManager().activeSpan() == null) {
            return NoopSpan.INSTANCE;
        }

        if (operationNameDatum != null && (operationNameDatum instanceof Datum)) {
            operationName = ((Datum) operationNameDatum).datum.toString();
        } else {
            operationName = defaultOperationName;
        }

        Tracer.SpanBuilder spanBuilder = tracer.buildSpan(operationName)
                .withTag(Tags.SPAN_KIND.getKey(), Tags.SPAN_KIND_CLIENT)
                .withTag(Tags.COMPONENT.getKey(), "java-rethinkDb")
                .withTag(Tags.DB_TYPE.getKey(), "rethinkDb");

        Span span = spanBuilder.start();
        return span;
    }

}
