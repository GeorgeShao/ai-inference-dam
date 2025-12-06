package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/georgeshao/ai-inference-dam/internal/dispatcher"
	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

type Handler struct {
	store      storage.Store
	dispatcher *dispatcher.Dispatcher
}

func NewHandler(store storage.Store, d *dispatcher.Dispatcher) *Handler {
	return &Handler{
		store:      store,
		dispatcher: d,
	}
}

func (h *Handler) CreateNamespace(c *fiber.Ctx) error {
	var req types.CreateNamespaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Invalid request body"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Name is required"})
	}

	// Check if namespace already exists
	existing, err := h.store.GetNamespace(c.Context(), req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to check namespace"})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(types.ErrorResponse{Error: "Namespace already exists"})
	}

	now := time.Now()
	record := &storage.NamespaceRecord{
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if req.Provider != nil {
		record.ProviderEndpoint = req.Provider.APIEndpoint
		record.ProviderAPIKey = req.Provider.APIKey
		record.ProviderModel = req.Provider.Model
		record.ProviderHeaders = req.Provider.Headers
	}

	if err := h.store.CreateNamespace(c.Context(), record); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to create namespace"})
	}

	resp := recordToNamespace(record)
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *Handler) GetNamespace(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Name is required"})
	}

	record, err := h.store.GetNamespace(c.Context(), name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get namespace"})
	}
	if record == nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Namespace not found"})
	}

	stats, err := h.store.GetNamespaceStats(c.Context(), name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get namespace stats"})
	}

	resp := recordToNamespace(record)
	resp.Stats = &types.NamespaceStats{
		TotalRequests: stats.TotalRequests,
		Queued:        stats.Queued,
		Processing:    stats.Processing,
		Completed:     stats.Completed,
		Failed:        stats.Failed,
	}

	return c.JSON(resp)
}

func (h *Handler) UpdateNamespace(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Name is required"})
	}

	var req types.UpdateNamespaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Invalid request body"})
	}

	existing, err := h.store.GetNamespace(c.Context(), name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get namespace"})
	}
	if existing == nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Namespace not found"})
	}

	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Provider != nil {
		existing.ProviderEndpoint = req.Provider.APIEndpoint
		existing.ProviderAPIKey = req.Provider.APIKey
		existing.ProviderModel = req.Provider.Model
		existing.ProviderHeaders = req.Provider.Headers
	}
	existing.UpdatedAt = time.Now()

	if err := h.store.UpdateNamespace(c.Context(), name, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to update namespace"})
	}

	resp := recordToNamespace(existing)
	return c.JSON(resp)
}

func (h *Handler) DeleteNamespace(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Name is required"})
	}

	if name == "default" {
		return c.Status(fiber.StatusForbidden).JSON(types.ErrorResponse{Error: "Cannot delete default namespace"})
	}

	deletedRequests, err := h.store.DeleteNamespace(c.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Namespace not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to delete namespace"})
	}

	return c.JSON(types.DeleteNamespaceResponse{
		Message:         "Namespace '" + name + "' deleted successfully",
		DeletedRequests: deletedRequests,
	})
}

func (h *Handler) ListNamespaces(c *fiber.Ctx) error {
	records, err := h.store.ListNamespaces(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to list namespaces"})
	}

	namespaces := make([]types.Namespace, len(records))
	for i, record := range records {
		namespaces[i] = recordToNamespace(record)
	}

	return c.JSON(namespaces)
}

// QueueChatCompletion handles POST /v1/chat/completions
func (h *Handler) QueueChatCompletion(c *fiber.Ctx) error {
	namespace := c.Get("X-Namespace", "default")

	ns, err := h.store.GetNamespace(c.Context(), namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get namespace"})
	}
	if ns == nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Namespace not found: " + namespace})
	}

	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Invalid request body"})
	}

	var headerEndpoint, headerAPIKey *string
	passthroughHeaders := make(map[string]string)

	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)

		switch k {
		case "X-Provider-Endpoint":
			headerEndpoint = &v
		case "X-Provider-Key":
			headerAPIKey = &v
		case "X-Namespace":
			// Already handled
		default:
			// Skip internal headers
			if !strings.HasPrefix(k, "X-") && k != "Content-Type" && k != "Content-Length" && k != "Host" && k != "User-Agent" && k != "Accept" && k != "Accept-Encoding" && k != "Connection" {
				passthroughHeaders[k] = v
			}
			// Also pass through Authorization header
			if k == "Authorization" {
				passthroughHeaders[k] = v
			}
		}
	})

	requestID := "req_" + uuid.New().String()
	now := time.Now()

	record := &storage.RequestRecord{
		ID:                 requestID,
		Namespace:          namespace,
		Status:             types.StatusQueued,
		RequestPayload:     payload,
		PassthroughHeaders: passthroughHeaders,
		HeaderEndpoint:     headerEndpoint,
		HeaderAPIKey:       headerAPIKey,
		CreatedAt:          now,
	}

	if err := h.store.CreateRequest(c.Context(), record); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to queue request"})
	}

	return c.Status(fiber.StatusAccepted).JSON(types.QueuedRequestResponse{
		ID:        requestID,
		Namespace: namespace,
		Status:    types.StatusQueued,
		CreatedAt: now.Format(time.RFC3339),
	})
}

func (h *Handler) GetRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "ID is required"})
	}

	record, err := h.store.GetRequest(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get request"})
	}
	if record == nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Request not found"})
	}

	return c.JSON(recordToRequest(record))
}

func (h *Handler) ListRequests(c *fiber.Ctx) error {
	namespace := c.Query("namespace")
	status := c.Query("status")
	cursor := c.Query("cursor")
	limit := c.QueryInt("limit", 100)

	filter := storage.RequestFilter{
		Limit: limit,
	}

	if namespace != "" {
		filter.Namespace = &namespace
	}
	if status != "" {
		s := types.RequestStatus(status)
		filter.Status = &s
	}
	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Invalid cursor format"})
		}
		filter.Cursor = &t
	}

	records, total, err := h.store.ListRequests(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to list requests"})
	}

	requests := make([]types.Request, len(records))
	for i, record := range records {
		requests[i] = recordToRequest(record)
	}

	// Set next cursor from last item's created_at if we have results
	var nextCursor *string
	if len(records) == limit {
		lastCreatedAt := records[len(records)-1].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &lastCreatedAt
	}

	return c.JSON(types.ListRequestsResponse{
		Requests:   requests,
		Total:      total,
		Limit:      limit,
		NextCursor: nextCursor,
	})
}

func (h *Handler) TriggerDispatch(c *fiber.Ctx) error {
	var req types.DispatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Invalid request body"})
	}

	if req.Namespace == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{Error: "Namespace is required"})
	}

	ns, err := h.store.GetNamespace(c.Context(), req.Namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get namespace"})
	}
	if ns == nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{Error: "Namespace not found"})
	}

	queuedRequests, err := h.store.GetQueuedRequests(c.Context(), req.Namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{Error: "Failed to get queued requests"})
	}

	if len(queuedRequests) == 0 {
		return c.Status(fiber.StatusOK).JSON(types.DispatchResponse{
			DispatchID:  "disp_" + uuid.New().String(),
			Namespace:   req.Namespace,
			QueuedCount: 0,
			Status:      "no_requests",
		})
	}

	dispatchID := "disp_" + uuid.New().String()
	go h.dispatcher.Dispatch(req.Namespace, dispatchID)

	return c.Status(fiber.StatusAccepted).JSON(types.DispatchResponse{
		DispatchID:  dispatchID,
		Namespace:   req.Namespace,
		QueuedCount: len(queuedRequests),
		Status:      "dispatching",
	})
}

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
