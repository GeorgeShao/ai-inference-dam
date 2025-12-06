package types

type DispatchRequest struct {
	Namespace string `json:"namespace"`
}

type DispatchResponse struct {
	DispatchID  string `json:"dispatch_id"`
	Namespace   string `json:"namespace"`
	QueuedCount int    `json:"queued_count"`
	Status      string `json:"status"`
}
