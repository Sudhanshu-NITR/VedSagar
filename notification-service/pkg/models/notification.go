package models

import "time"

type Notification struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	Recipient     string    `json:"recipient"`
	Channel       string    `json:"channel"` // "sms", "email", "push"
	Message       string    `json:"message"`
	Status        string    `json:"status"` // "pending", "success", "failed"
	Error         string    `json:"error,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Attempts      int       `json:"attempts"`
	MaxRetries    int       `json:"max_retries"`
	APIStatusCode int       `json:"api_status_code,omitempty"` // HTTP code from provider
	APIResponse   string    `json:"api_response,omitempty"`    // raw response body (short)
}

type DispatchResult struct {
	NotificationID string    `json:"notification_id"`
	Success        bool      `json:"success"`
	Error          string    `json:"error,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}
