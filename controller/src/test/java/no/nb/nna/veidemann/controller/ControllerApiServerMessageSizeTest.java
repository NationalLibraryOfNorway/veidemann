package no.nb.nna.veidemann.controller;

import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

import org.junit.jupiter.api.Test;

import io.grpc.ServerBuilder;

class ControllerApiServerMessageSizeTest {

    private final ServerBuilder<?> serverBuilder = mock(ServerBuilder.class);

    @Test
    void leavesGrpcDefaultWhenSizeIsUnset() {
        ControllerApiServer.configureMaxInboundMessageSize(serverBuilder, null);

        verify(serverBuilder, never()).maxInboundMessageSize(anyInt());
    }

    @Test
    void appliesConfiguredSize() {
        ControllerApiServer.configureMaxInboundMessageSize(serverBuilder, 16_777_216);

        verify(serverBuilder).maxInboundMessageSize(16_777_216);
    }
}
