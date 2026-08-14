package no.nb.nna.veidemann.frontier.db.script;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Function;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;

import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.exceptions.JedisConnectionException;
import redis.clients.jedis.exceptions.JedisDataException;

class RedisJobTest {

    @AfterEach
    void clearInterruptedFlag() {
        Thread.interrupted();
    }

    @Test
    void retriesReadOnlyResponseWithoutClosingSharedClient() {
        UnifiedJedis redisClient = mock(UnifiedJedis.class);
        AtomicInteger attempts = new AtomicInteger();
        TestRedisJob<String> job = new TestRedisJob<>("test", 1, millis -> {
        });

        String result = job.run(RedisJob.RedisContext.forClient(redisClient), client -> {
            if (attempts.getAndIncrement() == 0) {
                throw new JedisDataException("READONLY You can't write against a read only replica");
            }
            return "ok";
        });

        assertThat(result).isEqualTo("ok");
        assertThat(attempts).hasValue(2);
        verify(redisClient, never()).close();
    }

    @Test
    void retriesConnectionFailureWithoutClosingSharedClient() {
        UnifiedJedis redisClient = mock(UnifiedJedis.class);
        AtomicInteger attempts = new AtomicInteger();
        TestRedisJob<String> job = new TestRedisJob<>("test", 1, millis -> {
        });

        String result = job.run(RedisJob.RedisContext.forClient(redisClient), client -> {
            if (attempts.getAndIncrement() == 0) {
                throw new JedisConnectionException("connection failed");
            }
            return "ok";
        });

        assertThat(result).isEqualTo("ok");
        assertThat(attempts).hasValue(2);
        verify(redisClient, never()).close();
    }

    @Test
    void propagatesOriginalExceptionWhenRetriesAreExhausted() {
        UnifiedJedis redisClient = mock(UnifiedJedis.class);
        JedisConnectionException failure = new JedisConnectionException("connection failed");
        TestRedisJob<String> job = new TestRedisJob<>("test", 1, millis -> {
        });

        assertThatThrownBy(() -> job.run(
                RedisJob.RedisContext.forClient(redisClient),
                client -> {
                    throw failure;
                }))
                .isSameAs(failure);

        verify(redisClient, never()).close();
    }

    @Test
    void interruptedRetryRestoresInterruptFlagAndPropagatesOriginalException() {
        UnifiedJedis redisClient = mock(UnifiedJedis.class);
        JedisConnectionException failure = new JedisConnectionException("connection failed");
        TestRedisJob<String> job = new TestRedisJob<>("test", 1, millis -> {
            throw new InterruptedException("interrupted");
        });

        assertThatThrownBy(() -> job.run(
                RedisJob.RedisContext.forClient(redisClient),
                client -> {
                    throw failure;
                }))
                .isSameAs(failure);

        assertThat(Thread.currentThread().isInterrupted()).isTrue();
        assertThat(failure.getSuppressed())
                .singleElement()
                .isInstanceOf(InterruptedException.class);
        verify(redisClient, never()).close();
    }

    private static final class TestRedisJob<R> extends RedisJob<R> {
        private TestRedisJob(String name, int maxAttempts, Sleeper sleeper) {
            super(name, maxAttempts, sleeper);
        }

        private R run(RedisContext context, Function<UnifiedJedis, R> function) {
            return execute(context, function);
        }
    }
}
