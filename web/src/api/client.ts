import type {
  Namespace,
  Request,
  ListRequestsResponse,
  CreateNamespaceRequest,
  UpdateNamespaceRequest,
  DispatchResponse,
  DeleteNamespaceResponse,
  ErrorResponse,
} from '@/types';

const API_BASE = '';  // Proxied by Vite

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error: ErrorResponse = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(response.status, error.error);
  }
  return response.json();
}

export async function listNamespaces(): Promise<Namespace[]> {
  const response = await fetch(`${API_BASE}/namespaces`);
  return handleResponse<Namespace[]>(response);
}

export async function getNamespace(name: string): Promise<Namespace> {
  const response = await fetch(`${API_BASE}/namespaces/${encodeURIComponent(name)}`);
  return handleResponse<Namespace>(response);
}

export async function createNamespace(data: CreateNamespaceRequest): Promise<Namespace> {
  const response = await fetch(`${API_BASE}/namespaces`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<Namespace>(response);
}

export async function updateNamespace(name: string, data: UpdateNamespaceRequest): Promise<Namespace> {
  const response = await fetch(`${API_BASE}/namespaces/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<Namespace>(response);
}

export async function deleteNamespace(name: string): Promise<DeleteNamespaceResponse> {
  const response = await fetch(`${API_BASE}/namespaces/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
  return handleResponse<DeleteNamespaceResponse>(response);
}

export interface ListRequestsParams {
  namespace?: string;
  status?: string;
  cursor?: string;
  limit?: number;
}

export async function listRequests(params: ListRequestsParams = {}): Promise<ListRequestsResponse> {
  const searchParams = new URLSearchParams();
  if (params.namespace) searchParams.set('namespace', params.namespace);
  if (params.status) searchParams.set('status', params.status);
  if (params.cursor) searchParams.set('cursor', params.cursor);
  if (params.limit) searchParams.set('limit', params.limit.toString());

  const response = await fetch(`${API_BASE}/requests?${searchParams.toString()}`);
  return handleResponse<ListRequestsResponse>(response);
}

export async function getRequest(id: string): Promise<Request> {
  const response = await fetch(`${API_BASE}/requests/${encodeURIComponent(id)}`);
  return handleResponse<Request>(response);
}

export async function triggerDispatch(namespace: string): Promise<DispatchResponse> {
  const response = await fetch(`${API_BASE}/dispatch`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ namespace }),
  });
  return handleResponse<DispatchResponse>(response);
}
