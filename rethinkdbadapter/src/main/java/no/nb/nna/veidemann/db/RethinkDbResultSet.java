package no.nb.nna.veidemann.db;

import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.commons.db.DbResultSet;

import java.util.Iterator;
import java.util.NoSuchElementException;
import java.util.Objects;
import java.util.Spliterator;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.function.Consumer;
import java.util.stream.Stream;
import java.util.stream.StreamSupport;

final class RethinkDbResultSet<T> implements DbResultSet<T> {
    private final Stream<T> stream;
    private final Result<T> result;
    private final Iterator<T> iterator;
    private final Consumer<RuntimeException> connectionFailureHandler;

    private RethinkDbResultSet(Result<T> result, Iterator<T> iterator,
                               Consumer<RuntimeException> connectionFailureHandler) {
        this.result = result;
        this.iterator = iterator;
        this.connectionFailureHandler = connectionFailureHandler;
        if (result != null) {
            ResultSpliterator<T> spliterator = new ResultSpliterator<>(result, connectionFailureHandler);
            this.stream = StreamSupport.stream(spliterator, false).onClose(spliterator::close);
        } else {
            this.stream = StreamSupport.stream(Spliterators.spliteratorUnknownSize(iterator, 0), false);
        }
    }

    static <T> DbResultSet<T> fromResult(Result<T> result) {
        return fromResult(result, ignored -> { });
    }

    static <T> DbResultSet<T> fromResult(Result<T> result, Consumer<RuntimeException> connectionFailureHandler) {
        return new RethinkDbResultSet<>(Objects.requireNonNull(result), null,
                Objects.requireNonNull(connectionFailureHandler));
    }

    static <T> DbResultSet<T> fromIterator(Iterator<T> iterator) {
        return new RethinkDbResultSet<>(null, Objects.requireNonNull(iterator), ignored -> { });
    }

    @Override
    public boolean hasNext() {
        if (result != null) {
            try {
                return result.hasNext();
            } catch (RuntimeException e) {
                connectionFailureHandler.accept(e);
                throw e;
            }
        }
        return iterator.hasNext();
    }

    @Override
    public T next(long timeout, TimeUnit unit) throws TimeoutException {
        if (result != null) {
            try {
                return result.next(timeout, unit);
            } catch (RuntimeException e) {
                connectionFailureHandler.accept(e);
                throw e;
            }
        }
        return iterator.next();
    }

    @Override
    public Stream<T> stream() {
        return stream;
    }

    @Override
    public void close() {
        stream.close();
    }

    private static final class ResultSpliterator<T> implements Spliterator<T>, AutoCloseable {
        private final Result<T> result;
        private final Consumer<RuntimeException> connectionFailureHandler;
        private boolean closed;

        private ResultSpliterator(Result<T> result, Consumer<RuntimeException> connectionFailureHandler) {
            this.result = result;
            this.connectionFailureHandler = connectionFailureHandler;
        }

        @Override
        public boolean tryAdvance(Consumer<? super T> action) {
            if (action == null) {
                throw new NullPointerException();
            }
            while (!closed && hasNext()) {
                T next;
                try {
                    next = result.next(2000, TimeUnit.MILLISECONDS);
                } catch (TimeoutException e) {
                    continue;
                } catch (NoSuchElementException e) {
                    closed = true;
                    return false;
                } catch (RuntimeException e) {
                    connectionFailureHandler.accept(e);
                    throw e;
                }
                action.accept(next);
                return true;
            }
            return false;
        }

        private boolean hasNext() {
            try {
                return result.hasNext();
            } catch (RuntimeException e) {
                connectionFailureHandler.accept(e);
                throw e;
            }
        }

        @Override
        public Spliterator<T> trySplit() {
            return null;
        }

        @Override
        public long estimateSize() {
            return Long.MAX_VALUE;
        }

        @Override
        public int characteristics() {
            return 0;
        }

        @Override
        public void close() {
            closed = true;
            result.close();
        }
    }

    private static final class Spliterators {
        private Spliterators() {
        }

        private static <T> Spliterator<T> spliteratorUnknownSize(Iterator<T> iterator, int characteristics) {
            return java.util.Spliterators.spliteratorUnknownSize(iterator, characteristics);
        }
    }
}
