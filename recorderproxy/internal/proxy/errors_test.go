package proxy

import (
	"errors"
	"fmt"
	"testing"
)

func TestPhaseErrorPreservesPhaseAndCause(t *testing.T) {
	cause := errors.New("dial failed")
	err := fmt.Errorf("connect: %w", NewPhaseError(PhaseConnectDial, cause))

	if got := Phase(err); got != PhaseConnectDial {
		t.Fatalf("Phase() = %q, want %q", got, PhaseConnectDial)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("phase error does not preserve cause: %v", err)
	}
}
