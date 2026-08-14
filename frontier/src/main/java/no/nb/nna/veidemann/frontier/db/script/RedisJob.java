package no.nb.nna.veidemann.frontier.db.script;

import java.util.Objects;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Function;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.exceptions.JedisConnectionException;
import redis.clients.jedis.exceptions.JedisDataException;

public class RedisJob<R> {
    private static final Logger LOG = LoggerFactory.getLogger(RedisJob.class);

    private final String name;
    private final AtomicLong runTimeNanos = new AtomicLong();
    private final AtomicLong invocations = new AtomicLong();
    private final int maxAttempts;
    private final Sleeper sleeper;

    public RedisJob(String name) {
        this(name, 10);
    }

    public RedisJob(String name, int maxAttempts) {
        this(name, maxAttempts, Thread::sleep);
    }

    RedisJob(String name, int maxAttempts, Sleeper sleeper) {
        this.name = Objects.requireNonNull(name, "name");
        this.maxAttempts = maxAttempts;
        this.sleeper = Objects.requireNonNull(sleeper, "sleeper");
    }

    protected R execute(RedisContext ctx, Function<UnifiedJedis, R> job) {
        int attempts = 0;

        while (true) {
            try {
                long start = System.nanoTime();
                R result = job.apply(ctx.getClient());

                if (LOG.isDebugEnabled()) {
                    long total = runTimeNanos.addAndGet(System.nanoTime() - start);
                    long count = invocations.incrementAndGet();
                    if (count % 200 == 0) {
                        float avgMs = (total / (float) count) / 1_000_000f;
                        LOG.debug("Script {}: invocations={}, avg={}ms", name, count, avgMs);
                    }
                }
                return result;

            } catch (JedisDataException ex) {
                // Sentinel/replication failover window: client ends up on a replica or demoted
                // master.
                if (isReadOnly(ex)) {
                    attempts++;

                    if (attempts > maxAttempts) {
                        LOG.error("Redis is READONLY. Giving up after {} attempts", attempts, ex);
                        throw ex;
                    }

                    LOG.warn("Redis is READONLY (attempt {}/{}). Will retry in one second",
                            attempts, maxAttempts, ex);
                    sleepBeforeRetry(1000, ex);
                    continue;
                }
                throw ex;

            } catch (JedisConnectionException ex) {
                attempts++;

                if (attempts > maxAttempts) {
                    LOG.error("Failed connecting to Redis. Giving up after {} attempts", attempts, ex);
                    throw ex;
                }
                LOG.warn("Failed connecting to Redis (attempt {}/{}). Will retry in one second",
                        attempts, maxAttempts, ex);
                sleepBeforeRetry(1000, ex);
            }
        }
    }

    private static boolean isReadOnly(JedisDataException ex) {
        String msg = ex.getMessage();
        return msg != null && msg.startsWith("READONLY");
    }

    private void sleepBeforeRetry(long millis, RuntimeException root) {
        try {
            sleeper.sleep(millis);
        } catch (InterruptedException ie) {
            Thread.currentThread().interrupt();
            root.addSuppressed(ie);
            throw root;
        }
    }

    @FunctionalInterface
    interface Sleeper {
        void sleep(long millis) throws InterruptedException;
    }

    public static class RedisContext {
        private final UnifiedJedis client;

        private RedisContext(UnifiedJedis client) {
            this.client = Objects.requireNonNull(client, "client");
        }

        public static RedisContext forClient(UnifiedJedis client) {
            return new RedisContext(client);
        }

        public UnifiedJedis getClient() {
            return client;
        }
    }
}
