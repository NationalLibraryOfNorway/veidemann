package no.nb.nna.veidemann.commons.db;

import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.stream.Stream;

public interface DbResultSet<T> extends AutoCloseable {
    boolean hasNext();

    T next(long timeout, TimeUnit unit) throws TimeoutException;

    Stream<T> stream();

    @Override
    void close();
}