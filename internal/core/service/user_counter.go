package service

import (
	"context"
	"log/slog"
	"time"

	"user-management-api/internal/core/port"
)

// UserCounter is the background concurrency task from the challenge: it
// periodically logs the total number of users in the database.
type UserCounter struct {
	service  port.UserService
	logger   *slog.Logger
	interval time.Duration
}

func NewUserCounter(service port.UserService, logger *slog.Logger, interval time.Duration) *UserCounter {
	return &UserCounter{service: service, logger: logger, interval: interval}
}

// Run blocks until ctx is cancelled, logging the user count every interval.
// Call it in its own goroutine.
func (c *UserCounter) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("user counter stopped")
			return
		case <-ticker.C:
			countCtx, cancel := context.WithTimeout(ctx, c.interval)
			count, err := c.service.Count(countCtx)
			cancel()
			if err != nil {
				c.logger.Error("counting users", "error", err)
				continue
			}
			c.logger.Info("total users in database", "count", count)
		}
	}
}
