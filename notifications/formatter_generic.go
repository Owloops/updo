package notifications

import (
	"encoding/json"
	"fmt"
)

type GenericFormatter struct{}

func (f *GenericFormatter) Format(payload WebhookPayload) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generic webhook payload: %w", err)
	}
	return data, nil
}
