package pebbledb

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

// Key prefixes
const (
	prefixNs    = "ns:"    // ns:{name} → namespace JSON
	prefixReq   = "req:"   // req:{id} → request JSON
	prefixSt    = "st:"    // st:{ns}:{status}:{ts}:{id} → empty
	prefixCount = "count:" // count:{ns}:{status} → int64
)

type PebbleStore struct {
	db          *pebble.DB
	batchWriter *BatchWriter
	useBatch    bool
}

type namespaceData struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	ProviderEndpoint *string           `json:"provider_endpoint,omitempty"`
	ProviderAPIKey   *string           `json:"provider_api_key,omitempty"`
	ProviderModel    *string           `json:"provider_model,omitempty"`
	ProviderHeaders  map[string]string `json:"provider_headers,omitempty"`
	CreatedAt        int64             `json:"created_at"` // Unix nano
	UpdatedAt        int64             `json:"updated_at"` // Unix nano
}

type requestData struct {
	ID                 string                 `json:"id"`
	Namespace          string                 `json:"namespace"`
	Status             string                 `json:"status"`
	RequestPayload     map[string]interface{} `json:"request_payload"`
	PassthroughHeaders map[string]string      `json:"passthrough_headers,omitempty"`
	HeaderEndpoint     *string                `json:"header_endpoint,omitempty"`
	HeaderAPIKey       *string                `json:"header_api_key,omitempty"`
	ResponsePayload    map[string]interface{} `json:"response_payload,omitempty"`
	Error              *string                `json:"error,omitempty"`
	CreatedAt          int64                  `json:"created_at"` // Unix nano
	DispatchedAt       *int64                 `json:"dispatched_at,omitempty"`
	CompletedAt        *int64                 `json:"completed_at,omitempty"`
}

func New(dbPath string, useBatch bool) (*PebbleStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	opts := &pebble.Options{
		Merger: &pebble.Merger{
			Name: "int64_add",
			Merge: func(key, value []byte) (pebble.ValueMerger, error) {
				return &int64Merger{sum: decodeInt64(value)}, nil
			},
		},
	}

	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database: %w", err)
	}

	store := &PebbleStore{
		db:       db,
		useBatch: useBatch,
	}

	if useBatch {
		store.batchWriter = NewBatchWriter(db, DefaultBatchWriterConfig())
	}

	return store, nil
}

func (s *PebbleStore) Close() error {
	// Close batch writer first to flush remaining writes
	if s.batchWriter != nil {
		if err := s.batchWriter.Close(); err != nil {
			return fmt.Errorf("failed to close batch writer: %w", err)
		}
	}
	return s.db.Close()
}

func nsKey(name string) []byte {
	return []byte(prefixNs + name)
}

func reqKey(id string) []byte {
	return []byte(prefixReq + id)
}

func stKey(ns, status string, ts int64, id string) []byte {
	return []byte(fmt.Sprintf("%s%s:%s:%020d:%s", prefixSt, ns, status, ts, id))
}

func stPrefix(ns, status string) []byte {
	return []byte(fmt.Sprintf("%s%s:%s:", prefixSt, ns, status))
}

func countKey(ns, status string) []byte {
	return []byte(fmt.Sprintf("%s%s:%s", prefixCount, ns, status))
}

func encodeInt64(n int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(n))
	return b
}

func decodeInt64(b []byte) int64 {
	if len(b) != 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(b))
}

type int64Merger struct {
	sum int64
}

func (m *int64Merger) MergeNewer(value []byte) error {
	m.sum += decodeInt64(value)
	return nil
}

func (m *int64Merger) MergeOlder(value []byte) error {
	m.sum += decodeInt64(value)
	return nil
}

func (m *int64Merger) Finish(includesBase bool) ([]byte, io.Closer, error) {
	return encodeInt64(m.sum), nil, nil
}

func upperBound(prefix []byte) []byte {
	ub := make([]byte, len(prefix))
	copy(ub, prefix)
	for i := len(ub) - 1; i >= 0; i-- {
		if ub[i] < 0xff {
			ub[i]++
			return ub
		}
		ub[i] = 0
	}
	return append(ub, 0)
}

func (s *PebbleStore) CreateNamespace(ctx context.Context, ns *storage.NamespaceRecord) error {
	data := namespaceData{
		Name:             ns.Name,
		Description:      ns.Description,
		ProviderEndpoint: ns.ProviderEndpoint,
		ProviderAPIKey:   ns.ProviderAPIKey,
		ProviderModel:    ns.ProviderModel,
		ProviderHeaders:  ns.ProviderHeaders,
		CreatedAt:        ns.CreatedAt.UnixNano(),
		UpdatedAt:        ns.UpdatedAt.UnixNano(),
	}

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal namespace: %w", err)
	}

	return s.db.Set(nsKey(ns.Name), value, pebble.Sync)
}

func (s *PebbleStore) GetNamespace(ctx context.Context, name string) (*storage.NamespaceRecord, error) {
	value, closer, err := s.db.Get(nsKey(name))
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	defer closer.Close()

	var data namespaceData
	if err := json.Unmarshal(value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal namespace: %w", err)
	}

	return toNamespaceRecord(&data), nil
}

func (s *PebbleStore) UpdateNamespace(ctx context.Context, name string, ns *storage.NamespaceRecord) error {
	existing, err := s.GetNamespace(ctx, name)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("namespace not found: %s", name)
	}

	data := namespaceData{
		Name:             name,
		Description:      ns.Description,
		ProviderEndpoint: ns.ProviderEndpoint,
		ProviderAPIKey:   ns.ProviderAPIKey,
		ProviderModel:    ns.ProviderModel,
		ProviderHeaders:  ns.ProviderHeaders,
		CreatedAt:        existing.CreatedAt.UnixNano(),
		UpdatedAt:        ns.UpdatedAt.UnixNano(),
	}

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal namespace: %w", err)
	}

	return s.db.Set(nsKey(name), value, pebble.Sync)
}

func (s *PebbleStore) DeleteNamespace(ctx context.Context, name string) (int, error) {
	batch := s.db.NewBatch()
	defer batch.Close()

	deletedCount := 0

	// Delete all requests for this namespace by iterating status indexes
	for _, status := range []string{string(types.StatusQueued), string(types.StatusProcessing), string(types.StatusCompleted), string(types.StatusFailed)} {
		prefix := stPrefix(name, status)
		iter, err := s.db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: upperBound(prefix),
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create iterator: %w", err)
		}

		for iter.First(); iter.Valid(); iter.Next() {
			id := extractIDFromStKey(iter.Key())
			if id != "" {
				batch.Delete(reqKey(id), nil)
				batch.Delete(iter.Key(), nil)
				deletedCount++
			}
		}
		iter.Close()

		// Delete count key
		batch.Delete(countKey(name, status), nil)
	}

	// Delete namespace
	batch.Delete(nsKey(name), nil)

	if err := batch.Commit(pebble.Sync); err != nil {
		return 0, fmt.Errorf("failed to commit batch: %w", err)
	}

	return deletedCount, nil
}

func (s *PebbleStore) ListNamespaces(ctx context.Context) ([]*storage.NamespaceRecord, error) {
	var records []*storage.NamespaceRecord

	prefix := []byte(prefixNs)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var data namespaceData
		if err := json.Unmarshal(iter.Value(), &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal namespace: %w", err)
		}
		records = append(records, toNamespaceRecord(&data))
	}

	return records, nil
}

func (s *PebbleStore) GetNamespaceStats(ctx context.Context, name string) (*types.NamespaceStats, error) {
	stats := &types.NamespaceStats{}

	for _, status := range []types.RequestStatus{types.StatusQueued, types.StatusProcessing, types.StatusCompleted, types.StatusFailed} {
		count := s.getCount(name, string(status))
		switch status {
		case types.StatusQueued:
			stats.Queued = int(count)
		case types.StatusProcessing:
			stats.Processing = int(count)
		case types.StatusCompleted:
			stats.Completed = int(count)
		case types.StatusFailed:
			stats.Failed = int(count)
		}
		stats.TotalRequests += int(count)
	}

	return stats, nil
}

func (s *PebbleStore) getCount(ns, status string) int64 {
	value, closer, err := s.db.Get(countKey(ns, status))
	if err != nil {
		return 0
	}
	defer closer.Close()
	return decodeInt64(value)
}

func (s *PebbleStore) CreateRequest(ctx context.Context, req *storage.RequestRecord) error {
	data := requestData{
		ID:                 req.ID,
		Namespace:          req.Namespace,
		Status:             string(req.Status),
		RequestPayload:     req.RequestPayload,
		PassthroughHeaders: req.PassthroughHeaders,
		HeaderEndpoint:     req.HeaderEndpoint,
		HeaderAPIKey:       req.HeaderAPIKey,
		CreatedAt:          req.CreatedAt.UnixNano(),
	}

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if s.useBatch {
		// Queue writes to batch writer for batched commits
		s.batchWriter.Set(reqKey(req.ID), value)
		s.batchWriter.Set(stKey(req.Namespace, string(req.Status), data.CreatedAt, req.ID), nil)
		s.batchWriter.Merge(countKey(req.Namespace, string(req.Status)), encodeInt64(1))
		return nil
	}

	// Direct sync writes
	batch := s.db.NewBatch()
	defer batch.Close()
	batch.Set(reqKey(req.ID), value, nil)
	batch.Set(stKey(req.Namespace, string(req.Status), data.CreatedAt, req.ID), nil, nil)
	batch.Merge(countKey(req.Namespace, string(req.Status)), encodeInt64(1), nil)
	return batch.Commit(pebble.Sync)
}

func (s *PebbleStore) GetRequest(ctx context.Context, id string) (*storage.RequestRecord, error) {
	data, err := s.getRequestData(id)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	return toRequestRecord(data), nil
}

func (s *PebbleStore) getRequestData(id string) (*requestData, error) {
	value, closer, err := s.db.Get(reqKey(id))
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}
	defer closer.Close()

	var data requestData
	if err := json.Unmarshal(value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}
	return &data, nil
}

func (s *PebbleStore) ListRequests(ctx context.Context, filter storage.RequestFilter) ([]*storage.RequestRecord, int, error) {
	if filter.Namespace == nil {
		return nil, 0, fmt.Errorf("namespace is required")
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 100
	}

	// Determine which statuses to query
	var statuses []string
	if filter.Status != nil {
		statuses = []string{string(*filter.Status)}
	} else {
		statuses = []string{
			string(types.StatusQueued),
			string(types.StatusProcessing),
			string(types.StatusCompleted),
			string(types.StatusFailed),
		}
	}

	var allRecords []*storage.RequestRecord
	total := 0

	for _, status := range statuses {
		prefix := stPrefix(*filter.Namespace, status)
		iter, err := s.db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: upperBound(prefix),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create iterator: %w", err)
		}

		// If cursor provided, seek to it
		var cursorKey []byte
		if filter.Cursor != nil {
			cursorKey = stKey(*filter.Namespace, status, filter.Cursor.UnixNano(), "")
		}

		for iter.First(); iter.Valid(); iter.Next() {
			total++

			// Skip entries at or before cursor
			if cursorKey != nil && bytes.Compare(iter.Key(), cursorKey) <= 0 {
				continue
			}

			if len(allRecords) < limit {
				id := extractIDFromStKey(iter.Key())
				if id != "" {
					data, err := s.getRequestData(id)
					if err != nil {
						iter.Close()
						return nil, 0, err
					}
					if data != nil {
						allRecords = append(allRecords, toRequestRecord(data))
					}
				}
			}
		}
		iter.Close()
	}

	return allRecords, total, nil
}

func (s *PebbleStore) UpdateRequestStatus(ctx context.Context, id string, status types.RequestStatus, dispatchedAt time.Time) error {
	data, err := s.getRequestData(id)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("request not found: %s", id)
	}

	oldStatus := data.Status
	oldTs := data.CreatedAt

	data.Status = string(status)
	dispatchedNano := dispatchedAt.UnixNano()
	data.DispatchedAt = &dispatchedNano

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	batch.Set(reqKey(id), value, nil)
	batch.Delete(stKey(data.Namespace, oldStatus, oldTs, id), nil)
	batch.Set(stKey(data.Namespace, string(status), oldTs, id), nil, nil)
	batch.Merge(countKey(data.Namespace, oldStatus), encodeInt64(-1), nil)
	batch.Merge(countKey(data.Namespace, string(status)), encodeInt64(1), nil)

	return batch.Commit(pebble.Sync)
}

func (s *PebbleStore) UpdateRequestResponse(ctx context.Context, id string, response map[string]interface{}) error {
	data, err := s.getRequestData(id)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("request not found: %s", id)
	}

	oldStatus := data.Status
	oldTs := data.CreatedAt

	data.Status = string(types.StatusCompleted)
	data.ResponsePayload = response
	completedNano := time.Now().UnixNano()
	data.CompletedAt = &completedNano

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	batch.Set(reqKey(id), value, nil)
	batch.Delete(stKey(data.Namespace, oldStatus, oldTs, id), nil)
	batch.Set(stKey(data.Namespace, string(types.StatusCompleted), oldTs, id), nil, nil)
	batch.Merge(countKey(data.Namespace, oldStatus), encodeInt64(-1), nil)
	batch.Merge(countKey(data.Namespace, string(types.StatusCompleted)), encodeInt64(1), nil)

	return batch.Commit(pebble.Sync)
}

func (s *PebbleStore) UpdateRequestError(ctx context.Context, id string, errMsg string) error {
	data, err := s.getRequestData(id)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("request not found: %s", id)
	}

	oldStatus := data.Status
	oldTs := data.CreatedAt

	data.Status = string(types.StatusFailed)
	data.Error = &errMsg
	completedNano := time.Now().UnixNano()
	data.CompletedAt = &completedNano

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	batch.Set(reqKey(id), value, nil)
	batch.Delete(stKey(data.Namespace, oldStatus, oldTs, id), nil)
	batch.Set(stKey(data.Namespace, string(types.StatusFailed), oldTs, id), nil, nil)
	batch.Merge(countKey(data.Namespace, oldStatus), encodeInt64(-1), nil)
	batch.Merge(countKey(data.Namespace, string(types.StatusFailed)), encodeInt64(1), nil)

	return batch.Commit(pebble.Sync)
}

func (s *PebbleStore) GetQueuedRequests(ctx context.Context, namespace string) ([]*storage.RequestRecord, error) {
	prefix := stPrefix(namespace, string(types.StatusQueued))
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	var records []*storage.RequestRecord

	for iter.First(); iter.Valid(); iter.Next() {
		id := extractIDFromStKey(iter.Key())
		if id != "" {
			data, err := s.getRequestData(id)
			if err != nil {
				return nil, err
			}
			if data != nil {
				records = append(records, toRequestRecord(data))
			}
		}
	}

	return records, nil
}

// --- Conversion helpers ---

func toNamespaceRecord(data *namespaceData) *storage.NamespaceRecord {
	return &storage.NamespaceRecord{
		Name:             data.Name,
		Description:      data.Description,
		ProviderEndpoint: data.ProviderEndpoint,
		ProviderAPIKey:   data.ProviderAPIKey,
		ProviderModel:    data.ProviderModel,
		ProviderHeaders:  data.ProviderHeaders,
		CreatedAt:        time.Unix(0, data.CreatedAt),
		UpdatedAt:        time.Unix(0, data.UpdatedAt),
	}
}

func toRequestRecord(data *requestData) *storage.RequestRecord {
	record := &storage.RequestRecord{
		ID:                 data.ID,
		Namespace:          data.Namespace,
		Status:             types.RequestStatus(data.Status),
		RequestPayload:     data.RequestPayload,
		PassthroughHeaders: data.PassthroughHeaders,
		HeaderEndpoint:     data.HeaderEndpoint,
		HeaderAPIKey:       data.HeaderAPIKey,
		ResponsePayload:    data.ResponsePayload,
		Error:              data.Error,
		CreatedAt:          time.Unix(0, data.CreatedAt),
	}

	if data.DispatchedAt != nil {
		t := time.Unix(0, *data.DispatchedAt)
		record.DispatchedAt = &t
	}
	if data.CompletedAt != nil {
		t := time.Unix(0, *data.CompletedAt)
		record.CompletedAt = &t
	}

	return record
}

// extractIDFromStKey extracts the request ID from a status key
// Key format: st:{ns}:{status}:{ts}:{id}
func extractIDFromStKey(key []byte) string {
	parts := bytes.Split(key, []byte(":"))
	if len(parts) >= 5 {
		return string(parts[len(parts)-1])
	}
	return ""
}
