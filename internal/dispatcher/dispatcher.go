package dispatcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/pkg/types"
)

type Config struct {
	MaxWorkers        int
	RequestTimeout    time.Duration
	RequestsPerSecond float64
}

func DefaultConfig() Config {
	return Config{
		MaxWorkers:        10,
		RequestTimeout:    300 * time.Second,
		RequestsPerSecond: 10,
	}
}

type Dispatcher struct {
	store            storage.Store
	client           *Client
	config           Config
	mu               sync.Mutex
	activeDispatches map[string]bool
	rateLimiters     map[string]*rate.Limiter
}

func New(store storage.Store, config Config) *Dispatcher {
	return &Dispatcher{
		store:            store,
		client:           NewClient(config.RequestTimeout),
		config:           config,
		activeDispatches: make(map[string]bool),
		rateLimiters:     make(map[string]*rate.Limiter),
	}
}

func (d *Dispatcher) Dispatch(namespace string, dispatchID string) {
	ctx := context.Background()

	d.mu.Lock()
	if d.activeDispatches[namespace] {
		d.mu.Unlock()
		log.Printf("[%s] Dispatch already in progress for namespace: %s", dispatchID, namespace)
		return
	}
	d.activeDispatches[namespace] = true
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.activeDispatches, namespace)
		d.mu.Unlock()
	}()

	log.Printf("[%s] Starting dispatch for namespace: %s", dispatchID, namespace)

	ns, err := d.store.GetNamespace(ctx, namespace)
	if err != nil || ns == nil {
		log.Printf("[%s] Failed to get namespace %s: %v", dispatchID, namespace, err)
		return
	}

	requests, err := d.store.GetQueuedRequests(ctx, namespace)
	if err != nil {
		log.Printf("[%s] Failed to get queued requests: %v", dispatchID, err)
		return
	}

	if len(requests) == 0 {
		log.Printf("[%s] No queued requests for namespace: %s", dispatchID, namespace)
		return
	}

	log.Printf("[%s] Processing %d requests for namespace: %s", dispatchID, len(requests), namespace)

	limiter := d.getRateLimiter(namespace)

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, d.config.MaxWorkers)

	for _, req := range requests {
		req := req // Capture loop var
		g.Go(func() error {
			// Wait for rate limiter
			if err := limiter.Wait(ctx); err != nil {
				return err
			}

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			d.processRequest(ctx, ns, req, dispatchID)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Printf("[%s] Dispatch completed with errors: %v", dispatchID, err)
	} else {
		log.Printf("[%s] Dispatch completed successfully for namespace: %s", dispatchID, namespace)
	}
}

func (d *Dispatcher) processRequest(ctx context.Context, ns *storage.NamespaceRecord, req *storage.RequestRecord, dispatchID string) {
	endpoint := resolveEndpoint(ns, req.HeaderEndpoint)
	apiKey := resolveAPIKey(ns, req.HeaderAPIKey)

	if endpoint == "" {
		errMsg := "Missing required configuration: API endpoint"
		log.Printf("[%s] Request %s failed: %s", dispatchID, req.ID, errMsg)
		if err := d.store.UpdateRequestError(ctx, req.ID, errMsg); err != nil {
			log.Printf("[%s] Failed to update request error: %v", dispatchID, err)
		}
		return
	}

	if apiKey == "" {
		errMsg := "Missing required configuration: API key"
		log.Printf("[%s] Request %s failed: %s", dispatchID, req.ID, errMsg)
		if err := d.store.UpdateRequestError(ctx, req.ID, errMsg); err != nil {
			log.Printf("[%s] Failed to update request error: %v", dispatchID, err)
		}
		return
	}

	if err := d.store.UpdateRequestStatus(ctx, req.ID, types.StatusProcessing); err != nil {
		log.Printf("[%s] Failed to update request status: %v", dispatchID, err)
		return
	}

	headers := mergeHeaders(ns, req.PassthroughHeaders)

	payload := req.RequestPayload
	if ns.ProviderModel != nil {
		payload = cloneAndOverrideModel(req.RequestPayload, *ns.ProviderModel)
	}

	fullURL := endpoint + "/chat/completions"

	response, err := d.client.SendRequest(ctx, fullURL, apiKey, headers, payload)
	if err != nil {
		errMsg := fmt.Sprintf("Provider request failed: %v", err)
		log.Printf("[%s] Request %s failed: %s", dispatchID, req.ID, errMsg)
		if updateErr := d.store.UpdateRequestError(ctx, req.ID, errMsg); updateErr != nil {
			log.Printf("[%s] Failed to update request error: %v", dispatchID, updateErr)
		}
		return
	}

	if err := d.store.UpdateRequestResponse(ctx, req.ID, response); err != nil {
		log.Printf("[%s] Failed to update request response: %v", dispatchID, err)
		return
	}

	log.Printf("[%s] Request %s completed successfully", dispatchID, req.ID)
}

func (d *Dispatcher) getRateLimiter(namespace string) *rate.Limiter {
	d.mu.Lock()
	defer d.mu.Unlock()

	if limiter, ok := d.rateLimiters[namespace]; ok {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(d.config.RequestsPerSecond), 1)
	d.rateLimiters[namespace] = limiter
	return limiter
}

func resolveEndpoint(ns *storage.NamespaceRecord, headerValue *string) string {
	if ns.ProviderEndpoint != nil {
		return *ns.ProviderEndpoint
	}
	if headerValue != nil {
		return *headerValue
	}
	return ""
}

func resolveAPIKey(ns *storage.NamespaceRecord, headerValue *string) string {
	if ns.ProviderAPIKey != nil {
		return *ns.ProviderAPIKey
	}
	if headerValue != nil {
		return *headerValue
	}
	return ""
}

func mergeHeaders(ns *storage.NamespaceRecord, passthroughHeaders map[string]string) map[string]string {
	result := make(map[string]string)

	// Start with passthrough headers
	for k, v := range passthroughHeaders {
		result[k] = v
	}

	// Override with namespace headers (higher priority)
	if ns.ProviderHeaders != nil {
		for k, v := range ns.ProviderHeaders {
			result[k] = v
		}
	}

	return result
}

func cloneAndOverrideModel(payload map[string]interface{}, model string) map[string]interface{} {
	cloned := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		cloned[k] = v
	}
	cloned["model"] = model
	return cloned
}
