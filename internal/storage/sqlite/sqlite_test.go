package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

func setupTestStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sqlite_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}

	return store, cleanup
}

func TestNamespaceCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:        "test-namespace",
		Description: "Test description",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Get namespace
	retrieved, err := store.GetNamespace(ctx, "test-namespace")
	if err != nil {
		t.Fatalf("GetNamespace failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetNamespace returned nil")
	}
	if retrieved.Name != "test-namespace" {
		t.Errorf("Name mismatch: got %s, want test-namespace", retrieved.Name)
	}
	if retrieved.Description != "Test description" {
		t.Errorf("Description mismatch: got %s, want 'Test description'", retrieved.Description)
	}

	// Update namespace
	endpoint := "https://api.example.com/v1"
	retrieved.ProviderEndpoint = &endpoint
	retrieved.UpdatedAt = time.Now()

	err = store.UpdateNamespace(ctx, "test-namespace", retrieved)
	if err != nil {
		t.Fatalf("UpdateNamespace failed: %v", err)
	}

	// Verify update
	updated, err := store.GetNamespace(ctx, "test-namespace")
	if err != nil {
		t.Fatalf("GetNamespace after update failed: %v", err)
	}
	if updated.ProviderEndpoint == nil || *updated.ProviderEndpoint != endpoint {
		t.Errorf("ProviderEndpoint not updated correctly")
	}

	// List namespaces
	namespaces, err := store.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces failed: %v", err)
	}
	if len(namespaces) != 1 {
		t.Errorf("Expected 1 namespace, got %d", len(namespaces))
	}

	// Delete namespace
	deleted, err := store.DeleteNamespace(ctx, "test-namespace")
	if err != nil {
		t.Fatalf("DeleteNamespace failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted requests, got %d", deleted)
	}

	// Verify deletion
	retrieved, err = store.GetNamespace(ctx, "test-namespace")
	if err != nil {
		t.Fatalf("GetNamespace after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Namespace should have been deleted")
	}
}

func TestNamespaceWithProviderConfig(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	endpoint := "https://api.openai.com/v1"
	apiKey := "sk-test-key"
	model := "gpt-4"
	headers := map[string]string{
		"OpenAI-Organization": "org-123",
	}

	ns := &storage.NamespaceRecord{
		Name:             "openai-test",
		Description:      "OpenAI namespace",
		ProviderEndpoint: &endpoint,
		ProviderAPIKey:   &apiKey,
		ProviderModel:    &model,
		ProviderHeaders:  headers,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	retrieved, err := store.GetNamespace(ctx, "openai-test")
	if err != nil {
		t.Fatalf("GetNamespace failed: %v", err)
	}

	if retrieved.ProviderEndpoint == nil || *retrieved.ProviderEndpoint != endpoint {
		t.Error("ProviderEndpoint mismatch")
	}
	if retrieved.ProviderAPIKey == nil || *retrieved.ProviderAPIKey != apiKey {
		t.Error("ProviderAPIKey mismatch")
	}
	if retrieved.ProviderModel == nil || *retrieved.ProviderModel != model {
		t.Error("ProviderModel mismatch")
	}
	if retrieved.ProviderHeaders["OpenAI-Organization"] != "org-123" {
		t.Error("ProviderHeaders mismatch")
	}
}

func TestRequestCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace first
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create request
	req := &storage.RequestRecord{
		ID:        "req_test123",
		Namespace: "test-ns",
		Status:    types.StatusQueued,
		RequestPayload: map[string]interface{}{
			"model": "gpt-4",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello!",
				},
			},
		},
		PassthroughHeaders: map[string]string{
			"Authorization": "Bearer test-token",
		},
		CreatedAt: now,
	}

	err = store.CreateRequest(ctx, req)
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	// Get request
	retrieved, err := store.GetRequest(ctx, "req_test123")
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetRequest returned nil")
	}
	if retrieved.ID != "req_test123" {
		t.Errorf("ID mismatch: got %s", retrieved.ID)
	}
	if retrieved.Status != types.StatusQueued {
		t.Errorf("Status mismatch: got %s", retrieved.Status)
	}

	// Update status
	dispatchedAt := time.Now()
	err = store.UpdateRequestStatus(ctx, "req_test123", types.StatusProcessing, dispatchedAt)
	if err != nil {
		t.Fatalf("UpdateRequestStatus failed: %v", err)
	}

	// Verify status update and dispatched_at
	retrieved, _ = store.GetRequest(ctx, "req_test123")
	if retrieved.Status != types.StatusProcessing {
		t.Errorf("Status not updated: got %s", retrieved.Status)
	}
	if retrieved.DispatchedAt == nil {
		t.Error("DispatchedAt should be set")
	} else if retrieved.DispatchedAt.Unix() != dispatchedAt.Unix() {
		t.Errorf("DispatchedAt mismatch: got %v, want %v", retrieved.DispatchedAt.Unix(), dispatchedAt.Unix())
	}

	// Update with response
	response := map[string]interface{}{
		"id":      "chatcmpl-xyz",
		"object":  "chat.completion",
		"created": 1701784800,
		"choices": []interface{}{
			map[string]interface{}{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Hello! How can I help you?",
				},
			},
		},
	}

	err = store.UpdateRequestResponse(ctx, "req_test123", response)
	if err != nil {
		t.Fatalf("UpdateRequestResponse failed: %v", err)
	}

	// Verify response update
	retrieved, _ = store.GetRequest(ctx, "req_test123")
	if retrieved.Status != types.StatusCompleted {
		t.Errorf("Status should be completed: got %s", retrieved.Status)
	}
	if retrieved.ResponsePayload == nil {
		t.Error("ResponsePayload should not be nil")
	}
	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
}

func TestRequestError(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create request
	req := &storage.RequestRecord{
		ID:             "req_error123",
		Namespace:      "test-ns",
		Status:         types.StatusQueued,
		RequestPayload: map[string]interface{}{"model": "gpt-4"},
		CreatedAt:      now,
	}
	err = store.CreateRequest(ctx, req)
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	// Update with error
	err = store.UpdateRequestError(ctx, "req_error123", "Rate limit exceeded")
	if err != nil {
		t.Fatalf("UpdateRequestError failed: %v", err)
	}

	// Verify error update
	retrieved, _ := store.GetRequest(ctx, "req_error123")
	if retrieved.Status != types.StatusFailed {
		t.Errorf("Status should be failed: got %s", retrieved.Status)
	}
	if retrieved.Error == nil || *retrieved.Error != "Rate limit exceeded" {
		t.Error("Error message mismatch")
	}
}

func TestListRequests(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create multiple requests
	for i := 0; i < 5; i++ {
		req := &storage.RequestRecord{
			ID:             "req_" + string(rune('a'+i)),
			Namespace:      "test-ns",
			Status:         types.StatusQueued,
			RequestPayload: map[string]interface{}{"model": "gpt-4"},
			CreatedAt:      now.Add(time.Duration(i) * time.Second),
		}
		err = store.CreateRequest(ctx, req)
		if err != nil {
			t.Fatalf("CreateRequest failed: %v", err)
		}
	}

	// List all requests
	namespace := "test-ns"
	requests, total, err := store.ListRequests(ctx, storage.RequestFilter{
		Namespace: &namespace,
	})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected 5 total requests, got %d", total)
	}
	if len(requests) != 5 {
		t.Errorf("Expected 5 requests, got %d", len(requests))
	}

	// List with cursor pagination - first page
	requests, total, err = store.ListRequests(ctx, storage.RequestFilter{
		Namespace: &namespace,
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("ListRequests with pagination failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Total should still be 5, got %d", total)
	}
	if len(requests) != 2 {
		t.Errorf("Expected 2 requests with limit, got %d", len(requests))
	}

	// List with cursor pagination - second page using cursor from first page
	if len(requests) > 0 {
		cursor := requests[len(requests)-1].CreatedAt
		requests, total, err = store.ListRequests(ctx, storage.RequestFilter{
			Namespace: &namespace,
			Limit:     2,
			Cursor:    &cursor,
		})
		if err != nil {
			t.Fatalf("ListRequests with cursor failed: %v", err)
		}
		if total != 5 {
			t.Errorf("Total should still be 5, got %d", total)
		}
		if len(requests) != 2 {
			t.Errorf("Expected 2 requests on second page, got %d", len(requests))
		}
	}

	// List by status
	status := types.StatusQueued
	requests, _, err = store.ListRequests(ctx, storage.RequestFilter{
		Namespace: &namespace,
		Status:    &status,
	})
	if err != nil {
		t.Fatalf("ListRequests by status failed: %v", err)
	}
	if len(requests) != 5 {
		t.Errorf("Expected 5 queued requests, got %d", len(requests))
	}
}

func TestNamespaceStats(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create requests with different statuses
	statuses := []types.RequestStatus{
		types.StatusQueued,
		types.StatusQueued,
		types.StatusProcessing,
		types.StatusCompleted,
		types.StatusFailed,
	}

	for i, status := range statuses {
		req := &storage.RequestRecord{
			ID:             "req_" + string(rune('a'+i)),
			Namespace:      "test-ns",
			Status:         status,
			RequestPayload: map[string]interface{}{"model": "gpt-4"},
			CreatedAt:      now,
		}
		err = store.CreateRequest(ctx, req)
		if err != nil {
			t.Fatalf("CreateRequest failed: %v", err)
		}
	}

	// Get stats
	stats, err := store.GetNamespaceStats(ctx, "test-ns")
	if err != nil {
		t.Fatalf("GetNamespaceStats failed: %v", err)
	}

	if stats.TotalRequests != 5 {
		t.Errorf("TotalRequests: got %d, want 5", stats.TotalRequests)
	}
	if stats.Queued != 2 {
		t.Errorf("Queued: got %d, want 2", stats.Queued)
	}
	if stats.Processing != 1 {
		t.Errorf("Processing: got %d, want 1", stats.Processing)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed: got %d, want 1", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", stats.Failed)
	}
}

func TestGetQueuedRequests(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create mixed requests
	for i := 0; i < 3; i++ {
		req := &storage.RequestRecord{
			ID:             "req_queued_" + string(rune('a'+i)),
			Namespace:      "test-ns",
			Status:         types.StatusQueued,
			RequestPayload: map[string]interface{}{"model": "gpt-4"},
			CreatedAt:      now,
		}
		err = store.CreateRequest(ctx, req)
		if err != nil {
			t.Fatalf("CreateRequest failed: %v", err)
		}
	}

	// Create non-queued request
	req := &storage.RequestRecord{
		ID:             "req_completed",
		Namespace:      "test-ns",
		Status:         types.StatusCompleted,
		RequestPayload: map[string]interface{}{"model": "gpt-4"},
		CreatedAt:      now,
	}
	err = store.CreateRequest(ctx, req)
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	// Get queued requests
	queued, err := store.GetQueuedRequests(ctx, "test-ns")
	if err != nil {
		t.Fatalf("GetQueuedRequests failed: %v", err)
	}

	if len(queued) != 3 {
		t.Errorf("Expected 3 queued requests, got %d", len(queued))
	}

	for _, r := range queued {
		if r.Status != types.StatusQueued {
			t.Errorf("Expected queued status, got %s", r.Status)
		}
	}
}

func TestDeleteNamespaceWithRequests(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create namespace
	ns := &storage.NamespaceRecord{
		Name:      "test-ns",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.CreateNamespace(ctx, ns)
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create requests
	for i := 0; i < 3; i++ {
		req := &storage.RequestRecord{
			ID:             "req_" + string(rune('a'+i)),
			Namespace:      "test-ns",
			Status:         types.StatusQueued,
			RequestPayload: map[string]interface{}{"model": "gpt-4"},
			CreatedAt:      now,
		}
		err = store.CreateRequest(ctx, req)
		if err != nil {
			t.Fatalf("CreateRequest failed: %v", err)
		}
	}

	// Delete namespace
	deleted, err := store.DeleteNamespace(ctx, "test-ns")
	if err != nil {
		t.Fatalf("DeleteNamespace failed: %v", err)
	}

	if deleted != 3 {
		t.Errorf("Expected 3 deleted requests, got %d", deleted)
	}

	// Verify requests are deleted
	namespace := "test-ns"
	requests, total, err := store.ListRequests(ctx, storage.RequestFilter{Namespace: &namespace})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if total != 0 {
		t.Errorf("Expected 0 requests after delete, got %d", total)
	}
	if len(requests) != 0 {
		t.Errorf("Expected empty requests list, got %d", len(requests))
	}
}
