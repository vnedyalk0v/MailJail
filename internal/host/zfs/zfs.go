package zfs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/host/command"
)

type Client struct {
	Runner command.Runner
}

func New(runner command.Runner) Client {
	return Client{Runner: runner}
}

func (c Client) EnsureDataset(ctx context.Context, dataset string) error {
	if strings.TrimSpace(dataset) == "" {
		return errors.New("dataset is required")
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "zfs",
		Args:    []string{"list", "-H", "-o", "name", dataset},
		Timeout: 30 * time.Second,
	})
	if err == nil {
		return nil
	}

	_, createErr := c.Runner.Run(ctx, command.Spec{
		Path:    "zfs",
		Args:    []string{"create", "-p", dataset},
		Timeout: 2 * time.Minute,
	})
	if createErr != nil {
		return fmt.Errorf("create zfs dataset %s: %w", dataset, createErr)
	}

	return nil
}
