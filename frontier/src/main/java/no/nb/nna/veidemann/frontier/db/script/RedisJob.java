package no.nb.nna.veidemann.frontier.db.script;

import java.util.Objects;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Function;
import java.util.function.Supplier;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import redis.clients.jedis.Jedis;
import redis.clients.jedis.exceptions.JedisConnectionException;
import redis.clients.jedis.exceptions.JedisDataException;

public class RedisJob<R> {
    private static final Logger LOG = LoggerFactory.getLogger(RedisJob.class);

    private final String name;
    private final AtomicLong runTimeNanos = new AtomicLong();
    private final AtomicLong invocations = new AtomicLong();
    private final int maxAttempts;

    public RedisJob(String name) {
        this(name, 10);
    }

    public RedisJob(String name, int maxAttempts) {
        this.name = Objects.requireNonNull(name, "name");
        this.maxAttempts = maxAttempts;
    }

    protected R execute(JedisContext ctx, Function<Jedis, R> job) {
        int attempts = 0;

        while (true) {
            try {
                long start = System.nanoTime();
                R result = job.apply(ctx.getJedis());

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
                    ctx.invalidate(); // <-- key: drop connection, borrow fresh next time

                    if (attempts > maxAttempts) {
                        LOG.error("Redis is READONLY. Giving up after {} attempts", attempts, ex);
                        throw ex;
                    }

                    LOG.warn("Redis is READONLY (attempt {}/{}). Will retry in one second",
                            attempts, maxAttempts, ex);
                    sleepUninterruptibly(1000, ex);
                    continue;
                }
                throw ex;

            } catch (JedisConnectionException ex) {
                attempts++;
                ctx.invalidate(); // also drop broken connection

                if (attempts > maxAttempts) {
                    LOG.error("Failed connecting to Redis. Giving up after {} attempts", attempts, ex);
                    throw ex;
                }
                LOG.warn("Failed connecting to Redis (attempt {}/{}). Will retry in one second",
                        attempts, maxAttempts, ex);
                sleepUninterruptibly(1000, ex);
            }
        }
    }

    private static boolean isReadOnly(JedisDataException ex) {
        String msg = ex.getMessage();
        return msg != null && msg.startsWith("READONLY");
    }

    private static void sleepUninterruptibly(long millis, Exception root) {
        try {
            Thread.sleep(millis);
        } catch (InterruptedException ie) {
            Thread.currentThread().interrupt();
            root.addSuppressed(ie);
            // Preserve your current behavior: bubble up original exception
            if (root instanceof RuntimeException re)
                throw re;
            throw new RuntimeException(root);
        }
    }

    public static class JedisContext implements AutoCloseable {
        private final Supplier<Jedis> jedisSupplier;
        private Jedis jedis;

        private JedisContext(Supplier<Jedis> jedisSupplier) {
            this.jedisSupplier = Objects.requireNonNull(jedisSupplier, "jedisSupplier");
        }

        public static JedisContext forSupplier(Supplier<Jedis> jedisSupplier) {
            return new JedisContext(jedisSupplier);
        }

        public Jedis getJedis() {
            if (jedis == null) {
                jedis = jedisSupplier.get();
            }
            return jedis;
        }

        /** Close current connection so next getJedis() borrows a fresh one. */
        public void invalidate() {
            if (jedis != null) {
                try {
                    jedis.close();
                } finally {
                    jedis = null;
                }
            }
        }

        @Override
        public void close() {
            invalidate();
        }
    }
}
