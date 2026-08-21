package notification

import (
	"context"
	"errors"
	"testing"
	"time"
)

type countingSender struct {
	calls     int
	failures  int
	permanent bool
}

func (s *countingSender) Send(context.Context, Message) error {
	s.calls++
	if s.calls <= s.failures {
		if s.permanent {
			return errors.Join(ErrPermanentRejection, errors.New("recipient opted out"))
		}
		return errors.New("provider unavailable")
	}
	return nil
}

func TestPermanentNotificationFailureStopsRetriesWithoutChangingTransientPolicy(t *testing.T) {
	now := time.Date(2026, 8, 21, 5, 0, 0, 0, time.UTC)
	permanent := &countingSender{failures: 4, permanent: true}
	router := New()
	if err := router.Register("sms", permanent); err != nil {
		t.Fatalf("register sms: %v", err)
	}
	dispatcher := Dispatcher{Router: router, MaxAttempts: 4, Now: func() time.Time { return now }}
	message := Message{ID: "notice-15", TenantID: "cargo-east", Recipient: "+860000", Channel: "sms", Body: "flight delayed", CreatedAt: now}
	attempts, err := dispatcher.Deliver(context.Background(), message)
	if !errors.Is(err, ErrPermanentRejection) {
		t.Fatalf("permanent delivery error = %v, want ErrPermanentRejection", err)
	}
	if permanent.calls != 1 || len(attempts) != 1 {
		t.Fatalf("permanent delivery calls=%d attempts=%d, want one", permanent.calls, len(attempts))
	}

	transient := &countingSender{failures: 2}
	router = New()
	if err := router.Register("email", transient); err != nil {
		t.Fatalf("register email: %v", err)
	}
	dispatcher.Router = router
	message.ID = "notice-15-retry"
	message.Channel = "email"
	attempts, err = dispatcher.Deliver(context.Background(), message)
	if err != nil {
		t.Fatalf("transient delivery: %v", err)
	}
	if transient.calls != 3 || len(attempts) != 3 {
		t.Fatalf("transient delivery calls=%d attempts=%d, want three", transient.calls, len(attempts))
	}
}
