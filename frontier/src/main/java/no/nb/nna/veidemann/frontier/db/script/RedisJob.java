package no.nb.nna.veidemann.frontier.db.script;

import java.util.Objects;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Function;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.exceptions.JedisConnectionException;

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
            } catch (JedisConnectionException ex) {
                attempts++;
                if (attempts > maxAttempts) {
                    LOG.error("Failed connecting to Redis. Giving up after {} attempts", attempts, ex);
                    throw ex;
                }
                LOG.warn("Failed connecting to Redis (attempt {}/{}). Will retry in one second",
                        attempts, maxAttempts, ex);
                try {
                    Thread.sleep(1000);
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    ex.addSuppressed(ie);
                    throw ex;
                }
            }
        }
    }

    /**
     * Class that wraps a Jedis connection.
     */
    public static class JedisContext implements AutoCloseable {
        private final JedisPool jedisPool;
        private Jedis jedis;

        private JedisContext(JedisPool jedisPool) {
            this.jedisPool = Objects.requireNonNull(jedisPool, "jedisPool");
        }

        public static JedisContext forPool(JedisPool jedisPool) {
            return new JedisContext(jedisPool);
        }

        public Jedis getJedis() {
            if (jedis == null) {
                jedis = jedisPool.getResource();
            }
            return jedis;
        }

        @Override
        public void close() {
            if (jedis != null) {
                jedis.close();
                jedis = null;
            }
        }
    }
}
