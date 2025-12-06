-- name: CreateNamespace :exec
INSERT INTO namespaces (name, description, provider_endpoint, provider_api_key, provider_model, provider_headers, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetNamespace :one
SELECT name, description, provider_endpoint, provider_api_key, provider_model, provider_headers, created_at, updated_at
FROM namespaces
WHERE name = ?;

-- name: UpdateNamespace :exec
UPDATE namespaces
SET description = ?, provider_endpoint = ?, provider_api_key = ?, provider_model = ?, provider_headers = ?, updated_at = ?
WHERE name = ?;

-- name: DeleteNamespace :exec
DELETE FROM namespaces WHERE name = ?;

-- name: ListNamespaces :many
SELECT name, description, provider_endpoint, provider_api_key, provider_model, provider_headers, created_at, updated_at
FROM namespaces
ORDER BY name;

-- name: CreateRequest :exec
INSERT INTO requests (id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetRequest :one
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE id = ?;

-- name: DeleteRequestsByNamespace :execrows
DELETE FROM requests WHERE namespace = ?;

-- name: UpdateRequestStatus :exec
UPDATE requests SET status = ? WHERE id = ?;

-- name: UpdateRequestResponse :exec
UPDATE requests SET status = 'completed', response_payload = ?, completed_at = ? WHERE id = ?;

-- name: UpdateRequestError :exec
UPDATE requests SET status = 'failed', error = ?, completed_at = ? WHERE id = ?;

-- name: GetQueuedRequestsByNamespace :many
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE namespace = ? AND status = 'queued'
ORDER BY created_at ASC;

-- name: CountRequestsByNamespace :one
SELECT COUNT(*) as total FROM requests WHERE namespace = ?;

-- name: CountRequestsByNamespaceAndStatus :one
SELECT COUNT(*) as total FROM requests WHERE namespace = ? AND status = ?;

-- name: GetNamespaceStats :one
SELECT
    COUNT(*) as total_requests,
    SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END) as queued,
    SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END) as processing,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
FROM requests
WHERE namespace = ?;

-- name: ListRequestsByNamespace :many
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE namespace = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListRequestsByNamespaceWithCursor :many
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE namespace = ? AND created_at < ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListRequestsByNamespaceAndStatus :many
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE namespace = ? AND status = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListRequestsByNamespaceAndStatusWithCursor :many
SELECT id, namespace, status, request_payload, passthrough_headers, header_endpoint, header_api_key, response_payload, error, created_at, completed_at
FROM requests
WHERE namespace = ? AND status = ? AND created_at < ?
ORDER BY created_at DESC
LIMIT ?;
