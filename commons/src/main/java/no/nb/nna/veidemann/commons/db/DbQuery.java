package no.nb.nna.veidemann.commons.db;

import java.util.Objects;

public final class DbQuery {
    private final String serializedQuery;

    public DbQuery(String serializedQuery) {
        if (serializedQuery == null || serializedQuery.isBlank()) {
            throw new IllegalArgumentException("serializedQuery must not be blank");
        }
        this.serializedQuery = serializedQuery;
    }

    public String serializedQuery() {
        return serializedQuery;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (!(o instanceof DbQuery)) {
            return false;
        }
        DbQuery dbQuery = (DbQuery) o;
        return serializedQuery.equals(dbQuery.serializedQuery);
    }

    @Override
    public int hashCode() {
        return Objects.hash(serializedQuery);
    }

    @Override
    public String toString() {
        return "DbQuery{" +
                "serializedQuery='" + serializedQuery + '\'' +
                '}';
    }
}