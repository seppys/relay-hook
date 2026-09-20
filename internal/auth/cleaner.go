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
	_ = c.repo.RemoveExpiredKeys(ctx)
}
