package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/georgeshao/ai-inference-dam/internal/dispatcher"
	"github.com/georgeshao/ai-inference-dam/internal/storage/sqlite"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

func setupTestApp(t *testing.T) (*fiber.App, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
		t.Fatalf("Failed to create store: %v", err)
	}

	d := dispatcher.New(store, dispatcher.DefaultConfig())

	app := fiber.New()
	SetupRoutes(app, store, d)

	cleanup := func() {
		// Wait for any in-flight dispatch goroutines to complete before closing the store
		d.Wait()
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}

	return app, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestCreateNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "test-ns", "description": "Test namespace"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	var ns types.Namespace
	if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if ns.Name != "test-ns" {
		t.Errorf("Name mismatch: got %s", ns.Name)
	}
	if ns.Description != "Test namespace" {
		t.Errorf("Description mismatch: got %s", ns.Description)
	}
}

func TestCreateNamespaceDuplicate(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	body := `{"name": "test-ns"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// First request should succeed
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("First create failed: %d", resp.StatusCode)
	}

	// Second request should fail
	req = httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestGetNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace first
	body := `{"name": "test-ns", "description": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Get namespace
	req = httptest.NewRequest(http.MethodGet, "/namespaces/test-ns", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var ns types.Namespace
	if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if ns.Name != "test-ns" {
		t.Errorf("Name mismatch: got %s", ns.Name)
	}
	if ns.Stats == nil {
		t.Error("Stats should be included")
	}
}

func TestGetNamespaceNotFound(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/namespaces/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUpdateNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "test-ns", "description": "Original"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Update namespace
	body = `{"description": "Updated"}`
	req = httptest.NewRequest(http.MethodPatch, "/namespaces/test-ns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var ns types.Namespace
	if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if ns.Description != "Updated" {
		t.Errorf("Description not updated: got %s", ns.Description)
	}
}

func TestDeleteNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "test-ns"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Delete namespace
	req = httptest.NewRequest(http.MethodDelete, "/namespaces/test-ns", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/namespaces/test-ns", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Namespace should be deleted")
	}
}

func TestDeleteDefaultNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create default namespace
	body := `{"name": "default"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Try to delete default namespace
	req = httptest.NewRequest(http.MethodDelete, "/namespaces/default", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestListNamespaces(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespaces
	for _, name := range []string{"ns1", "ns2", "ns3"} {
		body := `{"name": "` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		_, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
	}

	// List namespaces
	req := httptest.NewRequest(http.MethodGet, "/namespaces", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var namespaces []types.Namespace
	if err := json.NewDecoder(resp.Body).Decode(&namespaces); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(namespaces) != 3 {
		t.Errorf("Expected 3 namespaces, got %d", len(namespaces))
	}
}

func TestQueueChatCompletion(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create default namespace
	body := `{"name": "default"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Queue chat completion
	body = `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 202, got %d: %s", resp.StatusCode, string(respBody))
	}

	var queued types.QueuedRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&queued); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if queued.ID == "" {
		t.Error("Request ID should not be empty")
	}
	if queued.Namespace != "default" {
		t.Errorf("Namespace should be 'default', got %s", queued.Namespace)
	}
	if queued.Status != types.StatusQueued {
		t.Errorf("Status should be 'queued', got %s", queued.Status)
	}
}

func TestQueueChatCompletionWithNamespace(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "my-project"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Queue chat completion with namespace header
	body = `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Namespace", "my-project")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	var queued types.QueuedRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&queued); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if queued.Namespace != "my-project" {
		t.Errorf("Namespace should be 'my-project', got %s", queued.Namespace)
	}
}

func TestQueueChatCompletionNamespaceNotFound(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Namespace", "nonexistent")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetRequest(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace and queue request
	body := `{"name": "default"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	body = `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var queued types.QueuedRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&queued); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Get request
	req = httptest.NewRequest(http.MethodGet, "/requests/"+queued.ID, nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var request types.Request
	if err := json.NewDecoder(resp.Body).Decode(&request); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if request.ID != queued.ID {
		t.Errorf("ID mismatch")
	}
}

func TestListRequests(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "default"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Queue multiple requests
	for i := 0; i < 3; i++ {
		body = `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
		req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		_, err = app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
	}

	// List requests
	req = httptest.NewRequest(http.MethodGet, "/requests?namespace=default", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var listResp types.ListRequestsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if listResp.Total != 3 {
		t.Errorf("Expected 3 total requests, got %d", listResp.Total)
	}
	if len(listResp.Requests) != 3 {
		t.Errorf("Expected 3 requests, got %d", len(listResp.Requests))
	}
}

func TestTriggerDispatch(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace
	body := `{"name": "test-ns"}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Trigger dispatch with no requests
	body = `{"namespace": "test-ns"}`
	req = httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for empty dispatch, got %d", resp.StatusCode)
	}

	var dispatchResp types.DispatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&dispatchResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dispatchResp.QueuedCount != 0 {
		t.Errorf("Expected 0 queued count, got %d", dispatchResp.QueuedCount)
	}
	if dispatchResp.Status != "no_requests" {
		t.Errorf("Expected 'no_requests' status, got %s", dispatchResp.Status)
	}
}

func TestTriggerDispatchWithRequests(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create namespace with provider config
	body := `{"name": "test-ns", "provider": {"api_endpoint": "https://api.example.com/v1", "api_key": "test-key"}}`
	req := httptest.NewRequest(http.MethodPost, "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Queue request
	body = `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Namespace", "test-ns")
	_, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Trigger dispatch
	body = `{"namespace": "test-ns"}`
	req = httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	var dispatchResp types.DispatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&dispatchResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dispatchResp.QueuedCount != 1 {
		t.Errorf("Expected 1 queued count, got %d", dispatchResp.QueuedCount)
	}
	if dispatchResp.Status != "dispatching" {
		t.Errorf("Expected 'dispatching' status, got %s", dispatchResp.Status)
	}
}

func TestTriggerDispatchNamespaceNotFound(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	body := `{"namespace": "nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}
