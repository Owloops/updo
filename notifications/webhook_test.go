package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendWebhook(t *testing.T) {
	tests := []struct {
		name           string
		payload        WebhookPayload
		headers        []string
		responseStatus int
		expectError    bool
	}{
		{
			name: "successful webhook",
			payload: WebhookPayload{
				Event:          "target_down",
				Target:         "Test Site",
				URL:            "https://example.com",
				Timestamp:      time.Now().UTC(),
				ResponseTimeMs: 1500,
				StatusCode:     500,
				Error:          "Internal Server Error",
			},
			headers:        []string{"X-Custom: test"},
			responseStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "webhook returns error status",
			payload: WebhookPayload{
				Event:          "target_up",
				Target:         "Test Site",
				URL:            "https://example.com",
				Timestamp:      time.Now().UTC(),
				ResponseTimeMs: 200,
				StatusCode:     200,
			},
			headers:        nil,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedPayload WebhookPayload
			var receivedHeaders http.Header

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHeaders = r.Header

				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}

				if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				w.WriteHeader(tc.responseStatus)
			}))
			defer server.Close()

			headerMap := make(map[string]string)
			for _, header := range tc.headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					headerMap[key] = value
				}
			}

			err := SendWebhook(server.URL, headerMap, tc.payload)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError {
				if receivedPayload.Event != tc.payload.Event {
					t.Errorf("Event mismatch: expected %s, got %s", tc.payload.Event, receivedPayload.Event)
				}
				if receivedPayload.Target != tc.payload.Target {
					t.Errorf("Target mismatch: expected %s, got %s", tc.payload.Target, receivedPayload.Target)
				}

				expectedHeaders := make(map[string]string)
				for _, header := range tc.headers {
					parts := strings.SplitN(header, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						expectedHeaders[key] = value
					}
				}

				for key, value := range expectedHeaders {
					if receivedHeaders.Get(key) != value {
						t.Errorf("Header %s mismatch: expected %s, got %s", key, value, receivedHeaders.Get(key))
					}
				}
			}
		})
	}
}

func TestHandleWebhookAlert(t *testing.T) {
	tests := []struct {
		name              string
		isUp              bool
		initialAlertSent  bool
		expectedAlertSent bool
		expectWebhookCall bool
		targetName        string
		targetURL         string
	}{
		{
			name:              "target goes down",
			isUp:              false,
			initialAlertSent:  false,
			expectedAlertSent: true,
			expectWebhookCall: true,
			targetName:        "Test Site",
			targetURL:         "https://example.com",
		},
		{
			name:              "target still down",
			isUp:              false,
			initialAlertSent:  true,
			expectedAlertSent: true,
			expectWebhookCall: false,
			targetName:        "Test Site",
			targetURL:         "https://example.com",
		},
		{
			name:              "target comes up",
			isUp:              true,
			initialAlertSent:  true,
			expectedAlertSent: false,
			expectWebhookCall: true,
			targetName:        "Test Site",
			targetURL:         "https://example.com",
		},
		{
			name:              "target still up",
			isUp:              true,
			initialAlertSent:  false,
			expectedAlertSent: false,
			expectWebhookCall: false,
			targetName:        "Test Site",
			targetURL:         "https://example.com",
		},
		{
			name:              "empty target name uses URL",
			isUp:              false,
			initialAlertSent:  false,
			expectedAlertSent: true,
			expectWebhookCall: true,
			targetName:        "",
			targetURL:         "https://example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			webhookCalled := false
			var receivedPayload WebhookPayload

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				webhookCalled = true
				if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
					t.Errorf("Failed to decode webhook payload: %v", err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			alertSent := tc.initialAlertSent

			HandleWebhookAlert(
				server.URL,
				nil,
				tc.isUp,
				&alertSent,
				tc.targetName,
				tc.targetURL,
				1500*time.Millisecond,
				200,
				"",
			)

			if alertSent != tc.expectedAlertSent {
				t.Errorf("Expected alertSent to be %v, got %v", tc.expectedAlertSent, alertSent)
			}

			if webhookCalled != tc.expectWebhookCall {
				t.Errorf("Expected webhook to be called: %v, but was: %v", tc.expectWebhookCall, webhookCalled)
			}

			if tc.expectWebhookCall && webhookCalled {
				expectedTarget := tc.targetName
				if expectedTarget == "" {
					expectedTarget = tc.targetURL
				}
				if receivedPayload.Target != expectedTarget {
					t.Errorf("Expected target %s, got %s", expectedTarget, receivedPayload.Target)
				}

				expectedEvent := "target_down"
				if tc.isUp {
					expectedEvent = "target_up"
				}
				if receivedPayload.Event != expectedEvent {
					t.Errorf("Expected event %s, got %s", expectedEvent, receivedPayload.Event)
				}
			}
		})
	}
}

func TestHandleWebhookAlertEmptyURL(t *testing.T) {
	alertSent := false
	webhookCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	HandleWebhookAlert(
		"",
		nil,
		false,
		&alertSent,
		"Test Site",
		"https://example.com",
		1500*time.Millisecond,
		500,
		"Server Error",
	)

	if webhookCalled {
		t.Error("Webhook should not be called when URL is empty")
	}

	if !alertSent {
		t.Error("Alert state should still be updated even without webhook URL")
	}
}
