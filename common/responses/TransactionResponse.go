package responses

import "time"

type TransactionResponse struct {
	TransactionId string    `json:"transaction_id"`
	Completed     bool      `json:"completed"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	Error         string    `json:"error,omitempty"`
}
