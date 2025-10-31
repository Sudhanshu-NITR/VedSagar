package models

// Event defines the structure of the incoming JSON payload
type Event struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Message    string   `json:"message"`
	Severity   string   `json:"severity"`
	Channels   []string `json:"channels"`
	Recipients []string `json:"recipients"`
}
