package channels

import (
	"context"
	"fmt"
	"os"
	"time"

	"notification-service/internal/logger"
	"notification-service/pkg/models"

	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSHandler sends SMS via Twilio
type SMSHandler struct {
	client       *twilio.RestClient
	fromPhoneNum string
}

// NewSMSHandler initializes SMSHandler with Twilio credentials from env vars
func NewSMSHandler() *SMSHandler {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_PHONE_NUMBER")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	return &SMSHandler{
		client:       client,
		fromPhoneNum: from,
	}
}

// Send sends SMS using Twilio API and returns DispatchResult
func (h *SMSHandler) Send(ctx context.Context, notif models.Notification) models.DispatchResult {
	params := &openapi.CreateMessageParams{}
	params.SetTo(notif.Recipient)
	params.SetFrom(h.fromPhoneNum)
	params.SetBody(notif.Message)

	resp, err := h.client.Api.CreateMessage(params)
	if err != nil {
		logger.Error(fmt.Errorf("[SMS] Error sending to %s: %w", notif.Recipient, err))
		return models.DispatchResult{
			NotificationID: notif.ID,
			Success:        false,
			Error:          err.Error(),
			Timestamp:      time.Now(),
		}
	}

	logger.Info(fmt.Sprintf("[SMS] Sent to %s, SID=%s", notif.Recipient, *resp.Sid))

	return models.DispatchResult{
		NotificationID: notif.ID,
		Success:        true,
		Error:          "",
		Timestamp:      time.Now(),
	}
}
