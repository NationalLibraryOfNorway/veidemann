package no.nb.nna.veidemann.frontier.settings;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatIllegalArgumentException;

import org.junit.jupiter.api.Test;

class SettingsTest {

    @Test
    void validatesMaxInboundMessageSize() {
        Settings settings = new Settings();

        assertThat(settings.getMaxInboundMessageSize()).isNull();
        settings.setMaxInboundMessageSize(16_777_216);
        assertThat(settings.getMaxInboundMessageSize()).isEqualTo(16_777_216);
        assertThatIllegalArgumentException().isThrownBy(() -> settings.setMaxInboundMessageSize(0));
        assertThatIllegalArgumentException().isThrownBy(() -> settings.setMaxInboundMessageSize(-1));
    }
}
