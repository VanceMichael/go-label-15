package notification

import (
	"errors"
	"fmt"
	"strings"
)

var ErrPermanentRejection = errors.New("notification permanently rejected")

type DeliveryError struct {
	Channel   string
	Recipient string
	Cause     error
	Permanent bool
}

func (e *DeliveryError) Error() string {
	if e == nil {
		return "notification delivery failed"
	}
	parts := make([]string, 0, 3)
	if e.Channel != "" {
		parts = append(parts, "channel "+e.Channel)
	}
	if e.Recipient != "" {
		parts = append(parts, "recipient "+e.Recipient)
	}
	if e.Cause != nil {
		parts = append(parts, e.Cause.Error())
	}
	if len(parts) == 0 {
		return "notification delivery failed"
	}
	return strings.Join(parts, ": ")
}

func (e *DeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Permanent {
		return errors.Join(ErrPermanentRejection, e.Cause)
	}
	return e.Cause
}

func WrapDeliveryError(message Message, cause error) error {
	if cause == nil {
		return nil
	}
	var existing *DeliveryError
	if errors.As(cause, &existing) {
		return cause
	}
	return &DeliveryError{
		Channel:   message.Channel,
		Recipient: message.Recipient,
		Cause:     fmt.Errorf("send message %s: %w", message.ID, cause),
		Permanent: errors.Is(cause, ErrPermanentRejection),
	}
}

func IsPermanent(err error) bool {
	return errors.Is(err, ErrPermanentRejection)
}
