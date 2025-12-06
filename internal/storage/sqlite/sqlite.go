package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/internal/storage/sqlite/sqlc"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

//go:embed schema.sql
var schemaSQL string

type SQLiteStore struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func New(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteStore{
		db:      db,
		queries: sqlc.New(db),
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(schemaSQL)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateNamespace(ctx context.Context, ns *storage.NamespaceRecord) error {
	headers, err := json.Marshal(ns.ProviderHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	return s.queries.CreateNamespace(ctx, sqlc.CreateNamespaceParams{
		Name:             ns.Name,
		Description:      ns.Description,
		ProviderEndpoint: toNullString(ns.ProviderEndpoint),
		ProviderApiKey:   toNullString(ns.ProviderAPIKey),
		ProviderModel:    toNullString(ns.ProviderModel),
		ProviderHeaders:  sql.NullString{String: string(headers), Valid: len(ns.ProviderHeaders) > 0},
		CreatedAt:        ns.CreatedAt.Unix(),
		UpdatedAt:        ns.UpdatedAt.Unix(),
	})
}

func (s *SQLiteStore) GetNamespace(ctx context.Context, name string) (*storage.NamespaceRecord, error) {
	ns, err := s.queries.GetNamespace(ctx, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return sqlcNamespaceToRecord(&ns)
}

func (s *SQLiteStore) UpdateNamespace(ctx context.Context, name string, ns *storage.NamespaceRecord) error {
	headers, err := json.Marshal(ns.ProviderHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	return s.queries.UpdateNamespace(ctx, sqlc.UpdateNamespaceParams{
		Name:             name,
		Description:      ns.Description,
		ProviderEndpoint: toNullString(ns.ProviderEndpoint),
		ProviderApiKey:   toNullString(ns.ProviderAPIKey),
		ProviderModel:    toNullString(ns.ProviderModel),
		ProviderHeaders:  sql.NullString{String: string(headers), Valid: len(ns.ProviderHeaders) > 0},
		UpdatedAt:        ns.UpdatedAt.Unix(),
	})
}

func (s *SQLiteStore) DeleteNamespace(ctx context.Context, name string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	deletedRequests, err := qtx.DeleteRequestsByNamespace(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("failed to delete requests: %w", err)
	}

	if err := qtx.DeleteNamespace(ctx, name); err != nil {
		return 0, fmt.Errorf("failed to delete namespace: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(deletedRequests), nil
}

func (s *SQLiteStore) ListNamespaces(ctx context.Context) ([]*storage.NamespaceRecord, error) {
	namespaces, err := s.queries.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	records := make([]*storage.NamespaceRecord, len(namespaces))
	for i, ns := range namespaces {
		record, err := sqlcNamespaceToRecord(&ns)
		if err != nil {
			return nil, err
		}
		records[i] = record
	}

	return records, nil
}

func (s *SQLiteStore) GetNamespaceStats(ctx context.Context, name string) (*types.NamespaceStats, error) {
	stats, err := s.queries.GetNamespaceStats(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace stats: %w", err)
	}

	return &types.NamespaceStats{
		TotalRequests: int(stats.TotalRequests),
		Queued:        nullFloat64ToInt(stats.Queued),
		Processing:    nullFloat64ToInt(stats.Processing),
		Completed:     nullFloat64ToInt(stats.Completed),
		Failed:        nullFloat64ToInt(stats.Failed),
	}, nil
}

func (s *SQLiteStore) CreateRequest(ctx context.Context, req *storage.RequestRecord) error {
	payload, err := json.Marshal(req.RequestPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	headers, err := json.Marshal(req.PassthroughHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal passthrough headers: %w", err)
	}

	return s.queries.CreateRequest(ctx, sqlc.CreateRequestParams{
		ID:                 req.ID,
		Namespace:          req.Namespace,
		Status:             string(req.Status),
		RequestPayload:     string(payload),
		PassthroughHeaders: sql.NullString{String: string(headers), Valid: len(req.PassthroughHeaders) > 0},
		HeaderEndpoint:     toNullString(req.HeaderEndpoint),
		HeaderApiKey:       toNullString(req.HeaderAPIKey),
		CreatedAt:          req.CreatedAt.Unix(),
	})
}

func (s *SQLiteStore) GetRequest(ctx context.Context, id string) (*storage.RequestRecord, error) {
	req, err := s.queries.GetRequest(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	return sqlcRequestToRecord(&req)
}

func (s *SQLiteStore) ListRequests(ctx context.Context, filter storage.RequestFilter) ([]*storage.RequestRecord, int, error) {
	limit := int64(filter.Limit)
	if limit == 0 {
		limit = 100 // Default limit
	}

	var requests []sqlc.Request
	var total int64
	var err error

	if filter.Namespace == nil {
		return nil, 0, fmt.Errorf("namespace is required")
	}

	if filter.Status != nil {
		if filter.Cursor != nil {
			requests, err = s.queries.ListRequestsByNamespaceAndStatusWithCursor(ctx, sqlc.ListRequestsByNamespaceAndStatusWithCursorParams{
				Namespace: *filter.Namespace,
				Status:    string(*filter.Status),
				CreatedAt: filter.Cursor.Unix(),
				Limit:     limit,
			})
		} else {
			requests, err = s.queries.ListRequestsByNamespaceAndStatus(ctx, sqlc.ListRequestsByNamespaceAndStatusParams{
				Namespace: *filter.Namespace,
				Status:    string(*filter.Status),
				Limit:     limit,
			})
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list requests: %w", err)
		}
		total, err = s.queries.CountRequestsByNamespaceAndStatus(ctx, sqlc.CountRequestsByNamespaceAndStatusParams{
			Namespace: *filter.Namespace,
			Status:    string(*filter.Status),
		})
	} else {
		if filter.Cursor != nil {
			requests, err = s.queries.ListRequestsByNamespaceWithCursor(ctx, sqlc.ListRequestsByNamespaceWithCursorParams{
				Namespace: *filter.Namespace,
				CreatedAt: filter.Cursor.Unix(),
				Limit:     limit,
			})
		} else {
			requests, err = s.queries.ListRequestsByNamespace(ctx, sqlc.ListRequestsByNamespaceParams{
				Namespace: *filter.Namespace,
				Limit:     limit,
			})
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list requests: %w", err)
		}
		total, err = s.queries.CountRequestsByNamespace(ctx, *filter.Namespace)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count requests: %w", err)
	}

	records := make([]*storage.RequestRecord, len(requests))
	for i, req := range requests {
		record, err := sqlcRequestToRecord(&req)
		if err != nil {
			return nil, 0, err
		}
		records[i] = record
	}

	return records, int(total), nil
}

func (s *SQLiteStore) UpdateRequestStatus(ctx context.Context, id string, status types.RequestStatus) error {
	return s.queries.UpdateRequestStatus(ctx, sqlc.UpdateRequestStatusParams{
		ID:     id,
		Status: string(status),
	})
}

func (s *SQLiteStore) UpdateRequestResponse(ctx context.Context, id string, response map[string]interface{}) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	return s.queries.UpdateRequestResponse(ctx, sqlc.UpdateRequestResponseParams{
		ID:              id,
		ResponsePayload: sql.NullString{String: string(responseJSON), Valid: true},
		CompletedAt:     sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
	})
}

func (s *SQLiteStore) UpdateRequestError(ctx context.Context, id string, errMsg string) error {
	return s.queries.UpdateRequestError(ctx, sqlc.UpdateRequestErrorParams{
		ID:          id,
		Error:       sql.NullString{String: errMsg, Valid: true},
		CompletedAt: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
	})
}

func (s *SQLiteStore) GetQueuedRequests(ctx context.Context, namespace string) ([]*storage.RequestRecord, error) {
	requests, err := s.queries.GetQueuedRequestsByNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get queued requests: %w", err)
	}

	records := make([]*storage.RequestRecord, len(requests))
	for i, req := range requests {
		record, err := sqlcRequestToRecord(&req)
		if err != nil {
			return nil, err
		}
		records[i] = record
	}

	return records, nil
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullFloat64ToInt(nf sql.NullFloat64) int {
	if !nf.Valid {
		return 0
	}
	return int(nf.Float64)
}

func sqlcNamespaceToRecord(ns *sqlc.Namespace) (*storage.NamespaceRecord, error) {
	record := &storage.NamespaceRecord{
		Name:             ns.Name,
		Description:      ns.Description,
		ProviderEndpoint: fromNullString(ns.ProviderEndpoint),
		ProviderAPIKey:   fromNullString(ns.ProviderApiKey),
		ProviderModel:    fromNullString(ns.ProviderModel),
		CreatedAt:        time.Unix(ns.CreatedAt, 0),
		UpdatedAt:        time.Unix(ns.UpdatedAt, 0),
	}

	if ns.ProviderHeaders.Valid && ns.ProviderHeaders.String != "" {
		if err := json.Unmarshal([]byte(ns.ProviderHeaders.String), &record.ProviderHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
	}

	return record, nil
}

func sqlcRequestToRecord(req *sqlc.Request) (*storage.RequestRecord, error) {
	record := &storage.RequestRecord{
		ID:             req.ID,
		Namespace:      req.Namespace,
		Status:         types.RequestStatus(req.Status),
		HeaderEndpoint: fromNullString(req.HeaderEndpoint),
		HeaderAPIKey:   fromNullString(req.HeaderApiKey),
		Error:          fromNullString(req.Error),
		CreatedAt:      time.Unix(req.CreatedAt, 0),
	}

	if req.CompletedAt.Valid {
		t := time.Unix(req.CompletedAt.Int64, 0)
		record.CompletedAt = &t
	}

	if err := json.Unmarshal([]byte(req.RequestPayload), &record.RequestPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request payload: %w", err)
	}

	if req.PassthroughHeaders.Valid && req.PassthroughHeaders.String != "" {
		if err := json.Unmarshal([]byte(req.PassthroughHeaders.String), &record.PassthroughHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal passthrough headers: %w", err)
		}
	}

	if req.ResponsePayload.Valid && req.ResponsePayload.String != "" {
		if err := json.Unmarshal([]byte(req.ResponsePayload.String), &record.ResponsePayload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response payload: %w", err)
		}
	}

	return record, nil
}
