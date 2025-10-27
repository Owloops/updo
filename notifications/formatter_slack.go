package notifications

import (
	"encoding/json"
	"fmt"
)

const (
	_eventTargetDown = "target_down"
	_eventTargetUp   = "target_up"
	_colorDanger     = "danger"
	_colorGood       = "good"
	_symbolDown      = "✘"
	_symbolUp        = "✔"
)

type slackMessage struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackFormatter struct{}

func (f *SlackFormatter) Format(payload WebhookPayload) ([]byte, error) {
	symbol := _symbolDown
	color := _colorDanger
	if payload.Event == _eventTargetUp {
		symbol = _symbolUp
		color = _colorGood
	}

	text := fmt.Sprintf("%s %s: %s", symbol, payload.Event, payload.Target)

	var fields []slackField

	fields = append(fields, slackField{
		Title: "URL",
		Value: payload.URL,
	})

	if payload.Error != "" {
		fields = append(fields, slackField{
			Title: "Error",
			Value: payload.Error,
		})
	}

	if payload.StatusCode > 0 {
		fields = append(fields, slackField{
			Title: "Status Code",
			Value: fmt.Sprintf("%d", payload.StatusCode),
			Short: true,
		})
	}

	fields = append(fields, slackField{
		Title: "Response Time",
		Value: fmt.Sprintf("%dms", payload.ResponseTimeMs),
		Short: true,
	})

	fields = append(fields, slackField{
		Title: "Timestamp",
		Value: payload.Timestamp.Format("2006-01-02 15:04:05 UTC"),
	})

	msg := slackMessage{
		Text: text,
		Attachments: []slackAttachment{
			{
				Color:  color,
				Fields: fields,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Slack webhook payload: %w", err)
	}

	return data, nil
}
