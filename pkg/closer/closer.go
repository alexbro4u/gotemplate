package closer

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Closer struct {
	mu    sync.Mutex
	funcs []func(context.Context) error
}

func New() *Closer {
	return &Closer{
		funcs: make([]func(context.Context) error, 0),
	}
}

func (c *Closer) Add(fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs = append(c.funcs, fn)
}

func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	funcs := make([]func(context.Context) error, len(c.funcs))
	copy(funcs, c.funcs)
	c.mu.Unlock()

	var errs []error
	for i := len(funcs) - 1; i >= 0; i-- {
		if err := funcs[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close: %w", errors.Join(errs...))
	}

	return nil
}
