package notifications

import (
	"strings"
)

type WebhookFormatter interface {
	Format(payload WebhookPayload) ([]byte, error)
}

func SelectFormatter(url string) WebhookFormatter {
	lowerURL := strings.ToLower(url)

	if strings.Contains(lowerURL, "hooks.slack.com") {
		return &SlackFormatter{}
	}

	if strings.Contains(lowerURL, "discord.com/api/webhooks") {
		return &DiscordFormatter{}
	}

	return &GenericFormatter{}
}
