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

    private RethinkDbResultSet(Result<T> result, Iterator<T> iterator) {
        this.result = result;
        this.iterator = iterator;
        if (result != null) {
            ResultSpliterator<T> spliterator = new ResultSpliterator<>(result);
            this.stream = StreamSupport.stream(spliterator, false).onClose(spliterator::close);
        } else {
            this.stream = StreamSupport.stream(Spliterators.spliteratorUnknownSize(iterator, 0), false);
        }
    }

    static <T> DbResultSet<T> fromResult(Result<T> result) {
        return new RethinkDbResultSet<>(Objects.requireNonNull(result), null);
    }

    static <T> DbResultSet<T> fromIterator(Iterator<T> iterator) {
        return new RethinkDbResultSet<>(null, Objects.requireNonNull(iterator));
    }

    @Override
    public boolean hasNext() {
        if (result != null) {
            return result.hasNext();
        }
        return iterator.hasNext();
    }

    @Override
    public T next(long timeout, TimeUnit unit) throws TimeoutException {
        if (result != null) {
            return result.next(timeout, unit);
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
        if (result != null) {
            result.close();
        }
    }

    private static final class ResultSpliterator<T> implements Spliterator<T>, AutoCloseable {
        private final Result<T> result;
        private boolean closed;

        private ResultSpliterator(Result<T> result) {
            this.result = result;
        }

        @Override
        public boolean tryAdvance(Consumer<? super T> action) {
            if (action == null) {
                throw new NullPointerException();
            }
            while (!closed && result.hasNext()) {
                try {
                    action.accept(result.next(2000, TimeUnit.MILLISECONDS));
                } catch (TimeoutException e) {
                    continue;
                } catch (NoSuchElementException e) {
                    if (closed || e.getMessage() != null && e.getMessage().contains("cancelled")) {
                        closed = true;
                        return false;
                    }
                    throw e;
                }
                return true;
            }
            return false;
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