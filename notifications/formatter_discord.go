package notifications

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	_discordColorRed   = 15158332
	_discordColorGreen = 3066993
)

type discordMessage struct {
	Content string         `json:"content"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title     string         `json:"title"`
	URL       string         `json:"url,omitempty"`
	Color     int            `json:"color"`
	Fields    []discordField `json:"fields,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordFormatter struct{}

func (f *DiscordFormatter) Format(payload WebhookPayload) ([]byte, error) {
	symbol := _symbolDown
	color := _discordColorRed
	if payload.Event == _eventTargetUp {
		symbol = _symbolUp
		color = _discordColorGreen
	}

	content := fmt.Sprintf("%s %s", symbol, payload.Event)

	var fields []discordField

	if payload.Error != "" {
		fields = append(fields, discordField{
			Name:  "Error",
			Value: payload.Error,
		})
	}

	if payload.StatusCode > 0 {
		fields = append(fields, discordField{
			Name:   "Status Code",
			Value:  fmt.Sprintf("%d", payload.StatusCode),
			Inline: true,
		})
	}

	fields = append(fields, discordField{
		Name:   "Response Time",
		Value:  fmt.Sprintf("%dms", payload.ResponseTimeMs),
		Inline: true,
	})

	msg := discordMessage{
		Content: content,
		Embeds: []discordEmbed{
			{
				Title:     payload.Target,
				URL:       payload.URL,
				Color:     color,
				Fields:    fields,
				Timestamp: payload.Timestamp,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Discord webhook payload: %w", err)
	}

	return data, nil
}
