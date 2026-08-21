package chromedp

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
)

func TestDOMEventConcurrentNodeMapRebuild(t *testing.T) {
	const (
		frameID     cdp.FrameID = "frame"
		nodeID      cdp.NodeID  = 1
		iterations              = 2_000
		nodesPerMap             = 64
	)

	frame := &cdp.Frame{
		ID: frameID,
		Nodes: map[cdp.NodeID]*cdp.Node{
			nodeID: {NodeID: nodeID},
		},
	}
	target := &Target{
		frames: map[cdp.FrameID]*cdp.Frame{frameID: frame},
		cur:    frameID,
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			frame.Lock()
			frame.Nodes = make(map[cdp.NodeID]*cdp.Node, nodesPerMap)
			for j := 1; j <= nodesPerMap; j++ {
				id := cdp.NodeID(j)
				frame.Nodes[id] = &cdp.Node{NodeID: id}
			}
			frame.Unlock()
			runtime.Gosched()
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			target.domEvent(context.Background(), &dom.EventAttributeModified{
				NodeID: nodeID,
				Name:   "data-race-regression",
				Value:  "safe",
			})
			runtime.Gosched()
		}
	}()

	close(start)
	wg.Wait()
}

func TestDOMEventWithoutCurrentFrame(t *testing.T) {
	target := &Target{frames: make(map[cdp.FrameID]*cdp.Frame)}
	target.domEvent(context.Background(), &dom.EventAttributeModified{
		NodeID: 1,
		Name:   "ignored",
		Value:  "ignored",
	})
}
