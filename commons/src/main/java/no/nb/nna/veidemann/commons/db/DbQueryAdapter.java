package no.nb.nna.veidemann.commons.db;

public interface DbQueryAdapter {
    DbResultSet<Object> executeQuery(DbQuery query) throws DbException;
}