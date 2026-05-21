package no.nb.nna.veidemann.db;

import com.rethinkdb.RethinkDB;
import com.rethinkdb.ast.ReqlAst;
import com.rethinkdb.gen.proto.TermType;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbQuery;
import no.nb.nna.veidemann.commons.db.DbQueryAdapter;
import no.nb.nna.veidemann.commons.db.DbResultSet;

public class RethinkDbQueryAdapter implements DbQueryAdapter {
    private final RethinkDbConnection conn;

    public RethinkDbQueryAdapter(RethinkDbConnection conn) {
        this.conn = conn;
    }

    @Override
    public DbResultSet<Object> executeQuery(DbQuery query) throws DbException {
        return conn.executeQuery("js-query", new RethinkPreparsedTerm(query.serializedQuery()));
    }

    private static class RethinkPreparsedTerm extends ReqlAst {
        private final String ast;

        protected RethinkPreparsedTerm(String ast) {
            super(TermType.DATUM, null, null);
            this.ast = ast;
        }

        @Override
        protected Object build() {
            try {
                return RethinkDB.getResultMapper().readValue(ast, Object.class);
            } catch (Exception e) {
                throw new RuntimeException(e);
            }
        }
    }
}