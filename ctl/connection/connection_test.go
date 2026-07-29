package connection

import (
	"testing"

	"github.com/spf13/viper"
)

func TestConnectRejectsMissingServerAddress(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("server", " \t\n")

	conn, err := connect()
	if conn != nil {
		conn.Close()
		t.Fatal("connect() returned a connection for an empty server address")
	}
	if err == nil {
		t.Fatal("connect() returned no error for an empty server address")
	}
	if err.Error() != "server address is not configured" {
		t.Fatalf("connect() error = %q, want %q", err, "server address is not configured")
	}
}
