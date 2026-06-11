package proxycompat

import (
	"context"
	"net"

	gerrors "github.com/getlantern/errors"
)

func (proxy *proxy) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return gerrors.New("Unable to accept: %v", err)
		}
		go func() {
			if err := proxy.Handle(context.Background(), conn, conn); err != nil {
				log.Errorf("Error handling connection: %v", err)
			}
		}()
	}
}
