import { useQuery } from '@tanstack/react-query';
import * as api from '@/api/client';
import type { ListRequestsParams } from '@/api/client';

const REQUEST_KEYS = {
  all: ['requests'] as const,
  list: (params: ListRequestsParams) => ['requests', 'list', params] as const,
  detail: (id: string) => ['requests', id] as const,
};

export function useRequests(params: ListRequestsParams = {}) {
  return useQuery({
    queryKey: REQUEST_KEYS.list(params),
    queryFn: () => api.listRequests(params),
    refetchInterval: 5000,  // Poll every 5 seconds
  });
}

export function useRequest(id: string) {
  return useQuery({
    queryKey: REQUEST_KEYS.detail(id),
    queryFn: () => api.getRequest(id),
    enabled: !!id,
  });
}
