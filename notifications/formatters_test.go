package notifications

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenericFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		payload WebhookPayload
		wantErr bool
	}{
		{
			name: _eventTargetDown + "_with_all_fields",
			payload: WebhookPayload{
				Event:          _eventTargetDown,
				Target:         "Test Service",
				URL:            testURL,
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 150,
				StatusCode:     500,
				Error:          "Internal Server Error",
			},
		},
		{
			name: _eventTargetUp + "_with_minimal_fields",
			payload: WebhookPayload{
				Event:          _eventTargetUp,
				Target:         "Test Service",
				URL:            testURL,
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &GenericFormatter{}
			data, err := f.Format(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenericFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal result: %v", err)
			}

			if result["event"] != tt.payload.Event {
				t.Errorf("event = %v, want %v", result["event"], tt.payload.Event)
			}
			if result["target"] != tt.payload.Target {
				t.Errorf("target = %v, want %v", result["target"], tt.payload.Target)
			}
		})
	}
}

func TestSlackFormatter_Format(t *testing.T) {
	tests := []struct {
		name      string
		payload   WebhookPayload
		wantErr   bool
		wantColor string
	}{
		{
			name: _eventTargetDown,
			payload: WebhookPayload{
				Event:          _eventTargetDown,
				Target:         "API Service",
				URL:            "https://api.example.com",
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 200,
				StatusCode:     503,
				Error:          "Service Unavailable",
			},
			wantColor: "danger",
		},
		{
			name: _eventTargetUp,
			payload: WebhookPayload{
				Event:          _eventTargetUp,
				Target:         "API Service",
				URL:            "https://api.example.com",
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 100,
			},
			wantColor: "good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &SlackFormatter{}
			data, err := f.Format(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var result slackMessage
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal result: %v", err)
				return
			}

			if len(result.Attachments) == 0 {
				t.Error("Expected attachments, got none")
				return
			}

			if result.Attachments[0].Color != tt.wantColor {
				t.Errorf("color = %v, want %v", result.Attachments[0].Color, tt.wantColor)
			}

			if result.Text == "" {
				t.Error("Expected non-empty text")
			}
		})
	}
}

func TestDiscordFormatter_Format(t *testing.T) {
	tests := []struct {
		name      string
		payload   WebhookPayload
		wantErr   bool
		wantColor int
	}{
		{
			name: _eventTargetDown,
			payload: WebhookPayload{
				Event:          _eventTargetDown,
				Target:         "Database",
				URL:            "https://db.example.com",
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 300,
				StatusCode:     0,
				Error:          "Connection timeout",
			},
			wantColor: _discordColorRed,
		},
		{
			name: _eventTargetUp,
			payload: WebhookPayload{
				Event:          _eventTargetUp,
				Target:         "Database",
				URL:            "https://db.example.com",
				Timestamp:      time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC),
				ResponseTimeMs: 50,
			},
			wantColor: _discordColorGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DiscordFormatter{}
			data, err := f.Format(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiscordFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var result discordMessage
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal result: %v", err)
				return
			}

			if len(result.Embeds) == 0 {
				t.Error("Expected embeds, got none")
				return
			}

			if result.Embeds[0].Color != tt.wantColor {
				t.Errorf("color = %v, want %v", result.Embeds[0].Color, tt.wantColor)
			}

			if result.Embeds[0].Title != tt.payload.Target {
				t.Errorf("title = %v, want %v", result.Embeds[0].Title, tt.payload.Target)
			}
		})
	}
}

func TestSelectFormatter(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType string
	}{
		{
			name:     "slack_webhook_standard",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
			wantType: slackFormatter,
		},
		{
			name:     "slack_webhook_uppercase",
			url:      "HTTPS://HOOKS.SLACK.COM/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
			wantType: slackFormatter,
		},
		{
			name:     "discord_webhook_standard",
			url:      "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz",
			wantType: discordFormatter,
		},
		{
			name:     "discord_webhook_uppercase",
			url:      "HTTPS://DISCORD.COM/API/WEBHOOKS/123456789012345678/abcdefghijklmnopqrstuvwxyz",
			wantType: discordFormatter,
		},
		{
			name:     "generic_webhook_custom",
			url:      testURL + "/webhook",
			wantType: genericFormatter,
		},
		{
			name:     "generic_webhook_localhost",
			url:      "http://localhost:8080/webhook",
			wantType: genericFormatter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := SelectFormatter(tt.url)
			formatterType := getFormatterType(formatter)
			if formatterType != tt.wantType {
				t.Errorf("SelectFormatter() = %v, want %v", formatterType, tt.wantType)
			}
		})
	}
}

func getFormatterType(f WebhookFormatter) string {
	switch f.(type) {
	case *SlackFormatter:
		return slackFormatter
	case *DiscordFormatter:
		return discordFormatter
	case *GenericFormatter:
		return genericFormatter
	default:
		return "unknown"
	}
}
