package utils

import (
	"log"

	"github.com/gen2brain/beeep"
)

func alert(message string) {
	err := beeep.Alert("Website Status", message, "assets/information.png")
	if err != nil {
		log.Printf("Failed to send alert: %v", err)
	}
}

func HandleAlerts(isUp bool, alertSent *bool) {
	if !isUp && !*alertSent {
		alert("The website is down!")
		*alertSent = true
	} else if isUp && *alertSent {
		alert("The website is back up!")
		*alertSent = false
	}
}
