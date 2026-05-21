package no.nb.nna.veidemann.commons.util;

import no.nb.nna.veidemann.commons.util.Pool.Lease;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Timeout;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.concurrent.atomic.AtomicInteger;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatIllegalStateException;
import static org.assertj.core.api.Assertions.assertThatExceptionOfType;

public class PoolTest {

    @Test
    @Timeout(10)
    public void lease() throws InterruptedException, TimeoutException, ExecutionException {
        AtomicInteger idx = new AtomicInteger(0);

        try (Pool<ExpensiveObject> pool = new Pool<>(4, () -> new ExpensiveObject(idx.getAndIncrement()), null, null)) {
            Lease<ExpensiveObject> l1 = null;
            Lease<ExpensiveObject> l2 = null;
            Lease<ExpensiveObject> l3 = null;
            Lease<ExpensiveObject> l4 = null;
            Lease<ExpensiveObject> l5 = null;
            Lease<ExpensiveObject> l6 = null;

            // Check that we can lease all objects in pool
            l1 = pool.lease();
            assertThat(l1.getObject().idx).isEqualTo(0);

            l2 = pool.lease();
            assertThat(l2.getObject().idx).isEqualTo(1);

            l3 = pool.lease();
            assertThat(l3.getObject().idx).isEqualTo(2);

            l4 = pool.lease();
            assertThat(l4.getObject().idx).isEqualTo(3);

            // Close one lease and check that we can't acces the object
            l3.close();
            Lease<ExpensiveObject> closedLease = l3;
            assertThatIllegalStateException()
                .isThrownBy(() -> { closedLease.getObject(); });

            // Check that we can lease the closed object
            l5 = pool.lease();
            assertThat(l5.getObject().idx).isEqualTo(2);

            // Check that we cannot lease object from exhausted pool.
            long start = System.currentTimeMillis();
            assertThatExceptionOfType(TimeoutException.class)
                .isThrownBy(() -> { pool.lease(500, TimeUnit.MILLISECONDS); });
            assertThat(System.currentTimeMillis() - start).isBetween(500L, 550L);

            ExecutorService executor = Executors.newSingleThreadExecutor();
            try {
                Future<Lease<ExpensiveObject>> l = executor.submit((Callable<Lease<ExpensiveObject>>) pool::lease);
                assertThatExceptionOfType(TimeoutException.class)
                    .isThrownBy(() -> l.get(200, TimeUnit.MILLISECONDS));
                assertThat(l.isDone()).isFalse();
                l1.close();
                l6 = l.get(1, TimeUnit.SECONDS);
                assertThat(l6.getObject().idx).isEqualTo(0);
                assertThat(l.isDone()).isTrue();
            } finally {
                executor.shutdownNow();
                if (l6 != null) {
                    l6.close();
                }
                if (l5 != null) {
                    l5.close();
                }
                if (l4 != null) {
                    l4.close();
                }
                if (l2 != null) {
                    l2.close();
                }
            }
        }

    }

    class ExpensiveObject {
        final int idx;

        public ExpensiveObject(int idx) {
            this.idx = idx;
        }

        @Override
        public String toString() {
            final StringBuffer sb = new StringBuffer("ExpensiveObject{");
            sb.append("idx=").append(idx);
            sb.append('}');
            return sb.toString();
        }
    }
}