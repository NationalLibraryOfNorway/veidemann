/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.frontier.worker;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.MoreExecutors;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.opentracing.contrib.grpc.TracingClientInterceptor;
import io.opentracing.util.GlobalTracer;
import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.dnsresolver.v1.DnsResolverGrpc;
import no.nb.nna.veidemann.api.dnsresolver.v1.ResolveReply;
import no.nb.nna.veidemann.api.dnsresolver.v1.ResolveRequest;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import no.nb.nna.veidemann.commons.client.GrpcUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.UnknownHostException;
import java.util.Objects;
import java.util.concurrent.TimeUnit;

/**
 *
 */
public class DnsServiceClient implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(DnsServiceClient.class);
    private static final int DNS_RCODE_NOERROR = 0;
    private static final int DNS_RCODE_NXDOMAIN = 3;

    private final ManagedChannel channel;
    private final DnsResolverGrpc.DnsResolverFutureStub futureStub;

    public DnsServiceClient(final String host, final int port) {
        this(ManagedChannelBuilder.forAddress(host, port).usePlaintext());
        LOG.info("DNS service client pointing to " + host + ":" + port);
    }

    public DnsServiceClient(ManagedChannelBuilder<?> channelBuilder) {
        LOG.debug("Setting up DNS service client");
        TracingClientInterceptor tracingInterceptor = TracingClientInterceptor.newBuilder().withTracer(GlobalTracer.get()).build();
        channel = channelBuilder.intercept(tracingInterceptor).build();
        futureStub = DnsResolverGrpc.newFutureStub(channel);
    }

    public ListenableFuture<Resolution> resolve(Frontier frontier, String host, int port, String executionId, ConfigRef collectionRef) {
        // Ensure host is never null
        String hostName = host == null ? "" : host;
        Objects.requireNonNull(collectionRef, "CollectionRef cannot be null");
        ResolveRequest request = ResolveRequest.newBuilder()
                .setHost(hostName)
                .setExecutionId(executionId)
                .setCollectionRef(collectionRef)
                .build();

        ListenableFuture<ResolveReply> reply = GrpcUtil.forkedCall(() -> futureStub.resolve(request));

        reply = Futures.catchingAsync(reply, Exception.class, e -> {
            if (e instanceof StatusRuntimeException) {
                StatusRuntimeException ex = (StatusRuntimeException) e;
                if (ex.getStatus().getCode() == Status.UNAVAILABLE.getCode()) {
                    LOG.error("RPC failed: " + ex.getStatus(), ex);
                } else {
                    LOG.debug("RPC failed: " + ex.getStatus());
                }
                UnknownHostException err = new UnknownHostException(hostName);
                err.initCause(ex);
                throw err;
            } else {
                throw e;
            }
        }, MoreExecutors.directExecutor());

        return Futures.transformAsync(reply, r -> {
            return Futures.immediateFuture(mapReply(hostName, port, r));
        }, frontier.getAsyncFunctionsThreadPool());
    }

    static Resolution mapReply(String requestedHost, int requestedPort, ResolveReply reply)
            throws UnknownHostException {
        if (reply.hasError()) {
            return Resolution.failure(mapDnsError(requestedHost, reply.getError()));
        }
        InetSocketAddress address = new InetSocketAddress(
                InetAddress.getByAddress(reply.getHost(), reply.getRawIp().toByteArray()), requestedPort);
        return Resolution.success(address);
    }

    static Error mapDnsError(String host, Error dnsError) {
        ExtraStatusCodes status;
        switch (dnsError.getCode()) {
            case DNS_RCODE_NOERROR:
                status = ExtraStatusCodes.DNS_NO_DATA;
                break;
            case DNS_RCODE_NXDOMAIN:
                status = ExtraStatusCodes.DNS_NXDOMAIN;
                break;
            default:
                status = ExtraStatusCodes.FAILED_DNS;
                break;
        }

        String reason = dnsError.getMsg().isBlank()
                ? "RCODE " + dnsError.getCode()
                : dnsError.getMsg();
        Error.Builder error = status.toFetchError(
                "DNS lookup for " + host + " returned " + reason + " (RCODE " + dnsError.getCode() + ")")
                .toBuilder();
        if (!dnsError.getDetail().isBlank()) {
            error.setDetail(dnsError.getDetail());
        } else {
            error.setDetail("DNS RCODE " + dnsError.getCode());
        }
        return error.build();
    }

    public static final class Resolution {
        private final InetSocketAddress address;
        private final Error error;

        private Resolution(InetSocketAddress address, Error error) {
            this.address = address;
            this.error = error;
        }

        static Resolution success(InetSocketAddress address) {
            return new Resolution(Objects.requireNonNull(address), null);
        }

        static Resolution failure(Error error) {
            return new Resolution(null, Objects.requireNonNull(error));
        }

        public boolean hasError() {
            return error != null;
        }

        public InetSocketAddress getAddress() {
            if (address == null) {
                throw new IllegalStateException("DNS resolution has no address");
            }
            return address;
        }

        public Error getError() {
            if (error == null) {
                throw new IllegalStateException("DNS resolution has no error");
            }
            return error;
        }
    }

    @Override
    public void close() {
        try {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        } catch (InterruptedException ex) {
            throw new RuntimeException(ex);
        }
    }


}
