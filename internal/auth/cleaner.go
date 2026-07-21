package auth

import (
	"context"
	"time"
)

type Cleaner struct {
	repo *Repository
}

func NewCleaner(repo *Repository) *Cleaner {
	return &Cleaner{repo: repo}
}

func (c *Cleaner) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.clean(ctx)
		}
	}
}

func (c *Cleaner) clean(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "auth.cleanExpiredKeys")
	defer span.End()

	if err := c.repo.RemoveExpiredKeys(ctx); err != nil {
		span.RecordError(err)
	}
}
