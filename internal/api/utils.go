package api

import (
	"time"

	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

func recordToNamespace(record *storage.NamespaceRecord) types.Namespace {
	ns := types.Namespace{
		Name:        record.Name,
		Description: record.Description,
		CreatedAt:   record.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   record.UpdatedAt.Format(time.RFC3339),
	}

	if record.ProviderEndpoint != nil || record.ProviderModel != nil || len(record.ProviderHeaders) > 0 {
		ns.Provider = &types.ProviderOverride{
			APIEndpoint: record.ProviderEndpoint,
			Model:       record.ProviderModel,
			Headers:     record.ProviderHeaders,
		}
	}

	return ns
}

func recordToRequest(record *storage.RequestRecord) types.Request {
	req := types.Request{
		ID:        record.ID,
		Namespace: record.Namespace,
		Status:    record.Status,
		CreatedAt: record.CreatedAt.Format(time.RFC3339),
	}

	if record.RequestPayload != nil {
		req.Request = record.RequestPayload
	}

	if record.DispatchedAt != nil {
		dispatchedAt := record.DispatchedAt.Format(time.RFC3339)
		req.DispatchedAt = &dispatchedAt
	}

	if record.CompletedAt != nil {
		completedAt := record.CompletedAt.Format(time.RFC3339)
		req.CompletedAt = &completedAt
	}

	if record.ResponsePayload != nil {
		req.Response = record.ResponsePayload
	}

	if record.Error != nil {
		req.Error = record.Error
	}

	return req
}
