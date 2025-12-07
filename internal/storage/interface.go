package storage

import (
	"context"
	"time"

	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

type Store interface {
	CreateNamespace(ctx context.Context, ns *NamespaceRecord) error
	GetNamespace(ctx context.Context, name string) (*NamespaceRecord, error)
	UpdateNamespace(ctx context.Context, name string, ns *NamespaceRecord) error
	DeleteNamespace(ctx context.Context, name string) (deletedRequests int, err error)
	ListNamespaces(ctx context.Context) ([]*NamespaceRecord, error)
	GetNamespaceStats(ctx context.Context, name string) (*types.NamespaceStats, error)

	CreateRequest(ctx context.Context, req *RequestRecord) error
	GetRequest(ctx context.Context, id string) (*RequestRecord, error)
	ListRequests(ctx context.Context, filter RequestFilter) ([]*RequestRecord, int, error)
	UpdateRequestStatus(ctx context.Context, id string, status types.RequestStatus, dispatchedAt time.Time) error
	UpdateRequestResponse(ctx context.Context, id string, response map[string]interface{}) error
	UpdateRequestError(ctx context.Context, id string, errMsg string) error
	GetQueuedRequests(ctx context.Context, namespace string) ([]*RequestRecord, error)

	Close() error
}
