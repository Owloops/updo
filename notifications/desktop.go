package notifications

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

func alert(message string) error {
	err := beeep.Alert("Website Status", message, "assets/information.png")
	return err
}

func HandleAlerts(isUp bool, alertSent *bool, targetName string, targetURL string) error {
	displayName := targetName
	if displayName == "" {
		displayName = targetURL
	}

	if !isUp && !*alertSent {
		err := alert(fmt.Sprintf("%s is down!", displayName))
		*alertSent = true
		if err != nil {
			return fmt.Errorf("failed to send alert: %w", err)
		}
	} else if isUp && *alertSent {
		err := alert(fmt.Sprintf("%s is back up!", displayName))
		*alertSent = false
		if err != nil {
			return fmt.Errorf("failed to send alert: %w", err)
		}
	}
	return nil
}
