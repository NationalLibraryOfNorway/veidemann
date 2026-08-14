package no.nb.nna.veidemann.frontier;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Duration;
import java.util.Set;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.Network;
import org.testcontainers.containers.wait.strategy.Wait;
import org.testcontainers.images.builder.Transferable;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import redis.clients.jedis.ConnectionPoolConfig;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.HostAndPort;
import redis.clients.jedis.RedisSentinelClient;

@Tag("integration")
@Tag("redis")
@Testcontainers
class RedisSentinelClientIntegrationTest {
    private static final String MASTER_NAME = "frontier-master";
    private static final Network NETWORK = Network.newNetwork();

    @SuppressWarnings("resource")
    @Container
    static final GenericContainer<?> REDIS_MASTER = new GenericContainer<>(DockerImageName.parse("redis:8-alpine"))
            .withNetwork(NETWORK)
            .withNetworkAliases("redis-master")
            .withExposedPorts(6379)
            .waitingFor(Wait.forLogMessage(".*Ready to accept connections.*", 1));

    @SuppressWarnings("resource")
    @Container
    static final GenericContainer<?> REDIS_SENTINEL = new GenericContainer<>(DockerImageName.parse("redis:8-alpine"))
            .withNetwork(NETWORK)
            .dependsOn(REDIS_MASTER)
            .withCopyToContainer(
                    Transferable.of("""
                            port 26379
                            protected-mode no
                            sentinel resolve-hostnames yes
                            sentinel monitor frontier-master redis-master 6379 1
                            sentinel down-after-milliseconds frontier-master 5000
                            sentinel failover-timeout frontier-master 10000
                            sentinel parallel-syncs frontier-master 1
                            """, 0777),
                    "/data/redis-sentinel.conf")
            .withCommand("redis-sentinel", "/data/redis-sentinel.conf")
            .withExposedPorts(26379)
            .waitingFor(Wait.forLogMessage(".*Sentinel ID is.*", 1)
                    .withStartupTimeout(Duration.ofSeconds(30)));

    @Test
    void discoversMasterAndExecutesCommands() {
        HostAndPort mappedMaster = new HostAndPort(REDIS_MASTER.getHost(), REDIS_MASTER.getFirstMappedPort());
        var masterConfig = DefaultJedisClientConfig.builder()
                .hostAndPortMapper(discovered -> mappedMaster)
                .build();
        var sentinelConfig = DefaultJedisClientConfig.builder()
                .serverDefaultProtocol()
                .build();

        try (RedisSentinelClient client = RedisSentinelClient.builder()
                .masterName(MASTER_NAME)
                .sentinels(Set.of(new HostAndPort(
                        REDIS_SENTINEL.getHost(), REDIS_SENTINEL.getFirstMappedPort())))
                .clientConfig(masterConfig)
                .sentinelClientConfig(sentinelConfig)
                .poolConfig(new ConnectionPoolConfig())
                .build()) {
            client.set("sentinel-smoke-test", "ok");

            assertThat(client.get("sentinel-smoke-test")).isEqualTo("ok");
            assertThat(client.getCurrentMaster()).isNotNull();
        }
    }
}
