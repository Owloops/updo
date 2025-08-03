package notifications

import (
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
)

func alert(message string) {
	err := beeep.Alert("Website Status", message, "assets/information.png")
	if err != nil {
		log.Printf("Failed to send alert: %v", err)
	}
}

func HandleAlerts(isUp bool, alertSent *bool, targetName string, targetURL string) {
	displayName := targetName
	if displayName == "" {
		displayName = targetURL
	}

	if !isUp && !*alertSent {
		alert(fmt.Sprintf("%s is down!", displayName))
		*alertSent = true
	} else if isUp && *alertSent {
		alert(fmt.Sprintf("%s is back up!", displayName))
		*alertSent = false
	}
}
