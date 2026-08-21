package notification

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

type DeliveryAttempt struct {
	MessageID string
	Attempt   int
	Error     error
	At        time.Time
}

type Dispatcher struct {
	Router      *Router
	MaxAttempts int
	Delay       time.Duration
	Now         func() time.Time
}

func (d Dispatcher) Deliver(ctx context.Context, message Message) ([]DeliveryAttempt, error) {
	if d.Router == nil {
		return nil, fmt.Errorf("notification router is required")
	}
	maxAttempts := d.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	attempts := make([]DeliveryAttempt, 0, maxAttempts)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return attempts, err
		}
		err := d.Router.Deliver(ctx, message)
		if err == nil {
			attempts = append(attempts, DeliveryAttempt{MessageID: message.ID, Attempt: attempt, At: d.now()})
			return attempts, nil
		}
		wrapped := WrapDeliveryError(message, err)
		aggregated := aggregateDeliveryErrors([]error{wrapped})
		attempts = append(attempts, DeliveryAttempt{MessageID: message.ID, Attempt: attempt, Error: aggregated, At: d.now()})
		if IsPermanent(aggregated) || attempt == maxAttempts {
			return attempts, aggregated
		}
		if err := waitDeliveryRetry(ctx, d.Delay); err != nil {
			return attempts, err
		}
	}
	return attempts, nil
}

func aggregateDeliveryErrors(failures []error) error {
	messages := make([]string, 0, len(failures))
	for _, failure := range failures {
		if failure != nil {
			messages = append(messages, failure.Error())
		}
	}
	if len(messages) == 0 {
		return nil
	}
	sort.Strings(messages)
	return errors.New("notification batch failed: " + messages[0])
}

func waitDeliveryRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (d Dispatcher) now() time.Time {
	if d.Now != nil {
		if now := d.Now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}
