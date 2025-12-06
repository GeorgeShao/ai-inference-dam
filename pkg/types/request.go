package types

type RequestStatus string

const (
	StatusQueued     RequestStatus = "queued"
	StatusProcessing RequestStatus = "processing"
	StatusCompleted  RequestStatus = "completed"
	StatusFailed     RequestStatus = "failed"
)

type Request struct {
	ID          string                 `json:"id"`
	Namespace   string                 `json:"namespace"`
	Status      RequestStatus          `json:"status"`
	Response    map[string]interface{} `json:"response,omitempty"`
	Error       *string                `json:"error,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	CompletedAt *string                `json:"completed_at,omitempty"`
}

type QueuedRequestResponse struct {
	ID        string        `json:"id"`
	Namespace string        `json:"namespace"`
	Status    RequestStatus `json:"status"`
	CreatedAt string        `json:"created_at"`
}

type ListRequestsResponse struct {
	Requests   []Request `json:"requests"`
	Total      int       `json:"total"`
	Limit      int       `json:"limit"`
	NextCursor *string   `json:"next_cursor,omitempty"`
}
