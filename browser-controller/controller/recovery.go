package controller

import (
	"context"
	"fmt"
	"time"

	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
)

// RecoverSlots runs once at process startup, before readiness or Frontier
// admission. The deployment must give this controller exclusive use of its
// recorder slots. Recovery failure leaves it unready and consumes no crawl work.
func RecoverSlots(ctx context.Context, recorder recorderV1.RecorderControlClient, slots int, timeout time.Duration) ([]string, error) {
	tokens := make([]string, slots)
	for slot := 1; slot <= slots; slot++ {
		recoveryCtx, cancel := context.WithTimeout(ctx, timeout)
		reply, err := recorder.RecoverSlot(recoveryCtx, &recorderV1.RecoverSlotRequest{ProxyId: int32(slot)})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("recover recorder slot %d: %w", slot, err)
		}
		if reply.GetNextToken() == "" {
			return nil, fmt.Errorf("recover recorder slot %d: missing admission token", slot)
		}
		tokens[slot-1] = reply.NextToken
	}
	return tokens, nil
}
