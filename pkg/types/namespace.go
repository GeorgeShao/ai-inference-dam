package types

type Namespace struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Provider    *ProviderOverride `json:"provider,omitempty"`
	Stats       *NamespaceStats   `json:"stats,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type ProviderOverride struct {
	APIEndpoint *string           `json:"api_endpoint,omitempty"`
	APIKey      *string           `json:"api_key,omitempty"`
	Model       *string           `json:"model,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type NamespaceStats struct {
	TotalRequests int `json:"total_requests"`
	Queued        int `json:"queued"`
	Processing    int `json:"processing"`
	Completed     int `json:"completed"`
	Failed        int `json:"failed"`
}

type CreateNamespaceRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Provider    *ProviderOverride `json:"provider,omitempty"`
}

type UpdateNamespaceRequest struct {
	Description *string           `json:"description,omitempty"`
	Provider    *ProviderOverride `json:"provider,omitempty"`
}

type DeleteNamespaceResponse struct {
	Message         string `json:"message"`
	DeletedRequests int    `json:"deleted_requests"`
}

type DeleteNamespaceConflictResponse struct {
	Error      string `json:"error"`
	Queued     int    `json:"queued"`
	Processing int    `json:"processing"`
	Completed  int    `json:"completed"`
}
