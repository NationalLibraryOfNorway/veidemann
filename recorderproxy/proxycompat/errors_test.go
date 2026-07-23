package proxycompat

import (
	"errors"
	"testing"
)

func TestUnavailableUpstreamConnReturnsOriginalFailure(t *testing.T) {
	cause := errors.New("upstream unavailable")
	conn := newUnavailableUpstreamConn(cause)

	if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, cause) {
		t.Fatalf("Read error = %v, want original failure", err)
	}
	if _, err := conn.Write([]byte("x")); !errors.Is(err, cause) {
		t.Fatalf("Write error = %v, want original failure", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}
	if conn.LocalAddr() == nil || conn.RemoteAddr() == nil {
		t.Fatal("adapter must provide addresses required by net.Conn")
	}
}

func TestPhaseErrorPreservesPhaseAndCause(t *testing.T) {
	cause := errors.New("dial failed")
	err := NewPhaseError(PhaseConnectDial, cause)

	if got := Phase(err); got != PhaseConnectDial {
		t.Fatalf("Phase() = %q, want %q", got, PhaseConnectDial)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("phase error does not preserve cause: %v", err)
	}
}
