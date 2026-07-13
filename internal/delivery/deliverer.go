package delivery

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

type Deliverer struct {
	client *http.Client
}

func NewDeliverer() *Deliverer {
	return &Deliverer{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *Deliverer) Deliver(ctx context.Context, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delivery to %s failed: status %d", endpoint, resp.StatusCode)
	}
	return nil
}
