package runner

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type Runner struct {
	g   *errgroup.Group
	ctx context.Context
}

func New(ctx context.Context) *Runner {
	g, _ := errgroup.WithContext(ctx)

	return &Runner{
		g:   g,
		ctx: ctx,
	}
}

func (r *Runner) Go(f func() error) {
	r.g.Go(f)
}

func (r *Runner) Wait() error {
	return r.g.Wait()
}
