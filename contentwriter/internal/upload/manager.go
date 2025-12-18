package upload

import "context"

type Manager struct {
	ch     chan string
	fn func(context.Context, string) error
}

func NewManager(buf int, upload func(context.Context, string) error) *Manager {
	return &Manager{
		ch:     make(chan string, buf),
		fn: upload,
	}
}

func (u *Manager) Enqueue(path string) {
	u.ch <- path
}

// Close signals “no more files”; Run will finish remaining queued paths and return.
func (u *Manager) Close() { close(u.ch) }

func (u *Manager) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err() // hard stop only
		case path, ok := <-u.ch:
			if !ok {
				return nil // drained
			}
			if err := u.fn(ctx, path); err != nil {
				return err
			}
		}
	}
}
