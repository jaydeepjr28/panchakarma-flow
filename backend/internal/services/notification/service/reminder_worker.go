package service

import (
	"context"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// ReminderWorker polls for due scheduled reminders and executes them.
type ReminderWorker struct {
	svc      NotificationService
	interval time.Duration
	stopCh   chan struct{}
}

// NewReminderWorker initializes the scheduled reminder polling worker.
func NewReminderWorker(svc NotificationService, interval time.Duration) *ReminderWorker {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &ReminderWorker{
		svc:      svc,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start runs the periodic check for due reminders in a background goroutine.
func (w *ReminderWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				count, err := w.svc.ProcessDueReminders(ctx)
				if err != nil {
					logger.Warn("Error processing due reminders", "error", err)
				} else if count > 0 {
					logger.Info("Processed and dispatched due reminders", "count", count)
				}
			}
		}
	}()
}

// Stop terminates the reminder polling loop.
func (w *ReminderWorker) Stop() {
	select {
	case <-w.stopCh:
		// already closed
	default:
		close(w.stopCh)
	}
}
