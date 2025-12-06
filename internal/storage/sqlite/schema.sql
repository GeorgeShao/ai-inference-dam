CREATE TABLE IF NOT EXISTS namespaces (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    provider_endpoint TEXT,
    provider_api_key TEXT,
    provider_model TEXT,
    provider_headers TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS requests (
    id TEXT PRIMARY KEY,
    namespace TEXT NOT NULL,
    status TEXT NOT NULL,
    request_payload TEXT NOT NULL,
    passthrough_headers TEXT,
    header_endpoint TEXT,
    header_api_key TEXT,
    response_payload TEXT,
    error TEXT,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    FOREIGN KEY (namespace) REFERENCES namespaces(name)
);

CREATE INDEX IF NOT EXISTS idx_requests_namespace_status ON requests(namespace, status);
CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
CREATE INDEX IF NOT EXISTS idx_requests_created_at ON requests(created_at);
