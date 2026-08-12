package no.nb.nna.veidemann.frontier.worker;

import static org.assertj.core.api.Assertions.assertThat;

import com.google.protobuf.ByteString;
import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.dnsresolver.v1.ResolveReply;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

class DnsServiceClientTest {

    @Test
    @SuppressWarnings("deprecation")
    void usesRequestedPortInsteadOfDeprecatedReplyPort() throws Exception {
        DnsServiceClient.Resolution resolution = DnsServiceClient.mapReply(
                "example.org",
                443,
                ResolveReply.newBuilder()
                        .setHost("example.org")
                        .setPort(1234)
                        .setRawIp(ByteString.copyFrom(new byte[]{127, 0, 0, 1}))
                        .build());

        assertThat(resolution.hasError()).isFalse();
        assertThat(resolution.getAddress().getPort()).isEqualTo(443);
    }

    @ParameterizedTest
    @CsvSource({
            "0, NODATA, DNS_NO_DATA",
            "3, NXDOMAIN, DNS_NXDOMAIN",
            "1, FORMERR, FAILED_DNS",
            "2, SERVFAIL, FAILED_DNS",
            "5, REFUSED, FAILED_DNS",
            "99, UNKNOWN, FAILED_DNS"
    })
    void mapsOnlyNxdomainAndNoDataToPermanentErrors(
            int dnsRcode,
            String dnsMessage,
            ExtraStatusCodes expectedStatus) {
        Error mapped = DnsServiceClient.mapDnsError(
                "example.org",
                Error.newBuilder()
                        .setCode(dnsRcode)
                        .setMsg(dnsMessage)
                        .setDetail("resolver detail")
                        .build());

        assertThat(mapped.getCode()).isEqualTo(expectedStatus.getCode());
        assertThat(mapped.getMsg()).contains("example.org", dnsMessage, "RCODE " + dnsRcode);
        assertThat(mapped.getDetail()).isEqualTo("resolver detail");
        assertThat(expectedStatus.isTemporary()).isEqualTo(expectedStatus == ExtraStatusCodes.FAILED_DNS);
    }
}
