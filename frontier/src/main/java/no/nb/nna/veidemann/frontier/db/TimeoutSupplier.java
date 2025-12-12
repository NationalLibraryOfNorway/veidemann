package no.nb.nna.veidemann.frontier.db;

import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.ReentrantLock;
import java.util.function.Consumer;
import java.util.function.Supplier;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.util.concurrent.ThreadFactoryBuilder;

public class TimeoutSupplier<E> implements AutoCloseable {
    private static final Logger LOG = LoggerFactory.getLogger(TimeoutSupplier.class);

    private final Supplier<E> supplier;
    private final Consumer<E> timeoutHandler;
    private final long elementTimeout;
    private final TimeUnit elementTimeoutUnit;

    private final BlockingQueue<Element<E>> queue;

    private final ScheduledExecutorService timeoutScheduler;
    private final ExecutorService supplierExecutor;

    private final ReentrantLock pauseLock = new ReentrantLock();
    private final Condition unpaused = pauseLock.newCondition();
    private volatile boolean paused = false;
    private volatile boolean running = true;

    public TimeoutSupplier(int capacity,
                           long timeout,
                           TimeUnit unit,
                           Supplier<E> supplier) {
        this(capacity, timeout, unit, 2, supplier, null);
    }

    public TimeoutSupplier(int capacity,
                           long timeout,
                           TimeUnit unit,
                           Supplier<E> supplier,
                           Consumer<E> timeoutHandler) {
        this(capacity, timeout, unit, 2, supplier, timeoutHandler);
    }

    public TimeoutSupplier(int capacity,
                           long timeout,
                           TimeUnit unit,
                           int workerThreads,
                           Supplier<E> supplier,
                           Consumer<E> timeoutHandler) {
        this.supplier = supplier;
        this.timeoutHandler = timeoutHandler;
        this.elementTimeout = timeout;
        this.elementTimeoutUnit = unit;

        this.queue = new ArrayBlockingQueue<>(capacity);

        this.timeoutScheduler = Executors.newSingleThreadScheduledExecutor(
                new ThreadFactoryBuilder()
                        .setNameFormat("TimeoutSupplierCleaner-%d")
                        .build()
        );

        this.supplierExecutor = Executors.newFixedThreadPool(
                workerThreads,
                new ThreadFactoryBuilder()
                        .setNameFormat("TimeoutSupplierWorker-%d")
                        .build()
        );

        for (int i = 0; i < workerThreads; i++) {
            supplierExecutor.submit(this::producerLoop);
        }
    }

    private void producerLoop() {
        try {
            while (running) {
                try {
                    awaitUnpausedOrStopped();
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    break;
                }

                if (!running) {
                    break;
                }

                E v;
                try {
                    v = supplier.get();
                } catch (Throwable t) {
                    LOG.warn("Error thrown by supplier function", t);
                    sleepQuietly(10, TimeUnit.MILLISECONDS);
                    continue;
                }

                if (v == null) {
                    continue; // no work this round
                }

                Element<E> e = new Element<>(v);
                ScheduledFuture<?> f = timeoutScheduler.schedule(
                        () -> expire(e),
                        elementTimeout,
                        elementTimeoutUnit
                );
                e.setTimeoutFuture(f);

                try {
                    queue.put(e); // back-pressure on capacity
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    break;
                }
            }
        } finally {
            // thread exits
        }
    }

    private void awaitUnpausedOrStopped() throws InterruptedException {
        pauseLock.lock();
        try {
            while (paused && running) {
                unpaused.await();
            }
        } finally {
            pauseLock.unlock();
        }
    }

    private void expire(Element<E> e) {
        if (!e.markExpired()) {
            return; // already claimed
        }

        queue.remove(e);

        if (timeoutHandler != null) {
            try {
                timeoutHandler.accept(e.value());
            } catch (Throwable t) {
                LOG.warn("Error in timeout handler", t);
            }
        }
    }

    public void pause(boolean pause) {
        pauseLock.lock();
        try {
            if (this.paused != pause) {
                this.paused = pause;
                if (!pause) {
                    unpaused.signalAll();
                }
            }
        } finally {
            pauseLock.unlock();
        }
    }

    /**
     * Get an element, waiting up to the given timeout.
     * Returns null on timeout or if stopped.
     */
    public E get(long timeout, TimeUnit unit) throws InterruptedException {
        long nanosTimeout = unit.toNanos(timeout);
        final long deadline = System.nanoTime() + nanosTimeout;

        while (true) {
            long remaining = deadline - System.nanoTime();
            if (remaining <= 0L) {
                return null;
            }

            Element<E> e = queue.poll(remaining, TimeUnit.NANOSECONDS);
            if (e == null) {
                return null; // timed out waiting for an element
            }

            E v = e.claim();
            if (v != null) {
                return v; // successfully claimed a non-expired element
            }

            // Already expired between poll and claim; loop and try again
        }
    }

    @Override
    public void close() throws InterruptedException {
        LOG.debug("Closing TimeoutSupplier");

        pauseLock.lock();
        try {
            running = false;
            paused = false;
            unpaused.signalAll();
        } finally {
            pauseLock.unlock();
        }

        // Drain remaining elements and treat them as timed out
        Element<E> e;
        while ((e = queue.poll()) != null) {
            expire(e);
        }

        supplierExecutor.shutdownNow();
        timeoutScheduler.shutdownNow();

        supplierExecutor.awaitTermination(5, TimeUnit.SECONDS);
        timeoutScheduler.awaitTermination(5, TimeUnit.SECONDS);

        LOG.debug("TimeoutSupplier closed");
    }

    private void sleepQuietly(long time, TimeUnit unit) {
        try {
            unit.sleep(time);
        } catch (InterruptedException ie) {
            Thread.currentThread().interrupt();
        }
    }

    private static final class Element<E> {
        private final E value;
        private final AtomicBoolean done = new AtomicBoolean(false);
        private volatile ScheduledFuture<?> timeoutFuture;

        Element(E value) {
            this.value = value;
        }

        void setTimeoutFuture(ScheduledFuture<?> timeoutFuture) {
            this.timeoutFuture = timeoutFuture;
        }

        E value() {
            return value;
        }

        E claim() {
            if (done.compareAndSet(false, true)) {
                ScheduledFuture<?> f = timeoutFuture;
                if (f != null) {
                    f.cancel(false);
                }
                return value;
            }
            return null;
        }

        boolean markExpired() {
            return done.compareAndSet(false, true);
        }
    }
}
