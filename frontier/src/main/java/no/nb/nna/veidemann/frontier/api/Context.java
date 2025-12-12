package no.nb.nna.veidemann.frontier.api;

import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

public class Context {

    private static final Logger LOG = LoggerFactory.getLogger(Context.class);

    private final Frontier frontier;

    // Lifecycle state guarded by lock
    private final Lock lock = new ReentrantLock();
    private final Condition terminationReached = lock.newCondition();

    private boolean shutdownRequested = false;
    private int activeObservers = 0;

    public Context(Frontier frontier) {
        this.frontier = frontier;
    }

    /**
     * Initiate shutdown. New observers should not be started,
     * and termination is signalled once all active observers have completed.
     */
    public void shutdown() {
        lock.lock();
        try {
            shutdownRequested = true;
            if (isTerminated()) {
                // No active observers – signal immediately
                terminationReached.signalAll();
            }
        } finally {
            lock.unlock();
        }
    }

    public boolean isShutdown() {
        lock.lock();
        try {
            return shutdownRequested;
        } finally {
            lock.unlock();
        }
    }

    /**
     * Await termination of this context.
     *
     * @param timeout timeout value
     * @param unit    timeout unit
     * @return true if terminated (shutdown requested and no active observers),
     *         false if the timeout elapsed before termination
     * @throws InterruptedException if interrupted while waiting
     */
    public boolean awaitTermination(long timeout, TimeUnit unit) throws InterruptedException {
        long nanos = unit.toNanos(timeout);
        lock.lock();
        try {
            while (!isTerminated()) {
                if (nanos <= 0L) {
                    return false; // timed out
                }
                nanos = terminationReached.awaitNanos(nanos);
            }
            return true;
        } finally {
            lock.unlock();
        }
    }

    /**
     * Await termination without timeout.
     *
     * @throws InterruptedException if interrupted while waiting
     */
    public void awaitTermination() throws InterruptedException {
        lock.lock();
        try {
            while (!isTerminated()) {
                terminationReached.await();
            }
        } finally {
            lock.unlock();
        }
    }

    public Frontier getFrontier() {
        return frontier;
    }

    public CrawlQueueManager getCrawlQueueManager() {
        return frontier.getCrawlQueueManager();
    }

    /**
     * Called when a streaming client starts (e.g. pageCompleted stream).
     */
    public void startPageComplete() {
        lock.lock();
        try {
            activeObservers++;
            LOG.trace("Client connected. Currently active clients: {}", activeObservers);
        } finally {
            lock.unlock();
        }
    }

    /**
     * Called when a streaming client completes.
     */
    public void setObserverCompleted() {
        lock.lock();
        try {
            if (activeObservers > 0) {
                activeObservers--;
            } else {
                LOG.warn("setObserverCompleted called but activeObservers is already 0");
            }

            if (isTerminated()) {
                terminationReached.signalAll();
            }

            LOG.trace("Client disconnected. Currently active clients: {}.", activeObservers);
        } finally {
            lock.unlock();
        }
    }

    /**
     * Terminated = shutdown requested AND no active observers.
     * Must be called under {@code lock}.
     */
    private boolean isTerminated() {
        return shutdownRequested && activeObservers <= 0;
    }
}
