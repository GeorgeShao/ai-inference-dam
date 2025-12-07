package storage

import (
	"time"

	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

type NamespaceRecord struct {
	Name             string
	Description      string
	ProviderEndpoint *string
	ProviderAPIKey   *string
	ProviderModel    *string
	ProviderHeaders  map[string]string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RequestRecord struct {
	ID                 string
	Namespace          string
	Status             types.RequestStatus
	RequestPayload     map[string]interface{}
	PassthroughHeaders map[string]string
	HeaderEndpoint     *string
	HeaderAPIKey       *string
	ResponsePayload    map[string]interface{}
	Error              *string
	CreatedAt          time.Time
	DispatchedAt       *time.Time
	CompletedAt        *time.Time
}

type RequestFilter struct {
	Namespace *string
	Status    *types.RequestStatus
	Limit     int
	Cursor    *time.Time // created_at cursor for pagination (get items before this time)
}
