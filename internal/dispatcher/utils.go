package dispatcher

import (
	"github.com/georgeshao/ai-inference-dam/internal/storage"
)

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
