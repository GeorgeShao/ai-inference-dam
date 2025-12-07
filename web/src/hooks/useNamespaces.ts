import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as api from '@/api/client';
import type { CreateNamespaceRequest, UpdateNamespaceRequest } from '@/types';

const NAMESPACE_KEYS = {
  all: ['namespaces'] as const,
  detail: (name: string) => ['namespaces', name] as const,
};

export function useNamespaces() {
  return useQuery({
    queryKey: NAMESPACE_KEYS.all,
    queryFn: api.listNamespaces,
    refetchInterval: 5000,  // Poll every 5 seconds
  });
}

export function useNamespace(name: string) {
  return useQuery({
    queryKey: NAMESPACE_KEYS.detail(name),
    queryFn: () => api.getNamespace(name),
    enabled: !!name,
  });
}

export function useCreateNamespace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateNamespaceRequest) => api.createNamespace(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NAMESPACE_KEYS.all });
    },
  });
}

export function useUpdateNamespace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, data }: { name: string; data: UpdateNamespaceRequest }) =>
      api.updateNamespace(name, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NAMESPACE_KEYS.all });
    },
  });
}

export function useDeleteNamespace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => api.deleteNamespace(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NAMESPACE_KEYS.all });
    },
  });
}

export function useDispatch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (namespace: string) => api.triggerDispatch(namespace),
    onSuccess: () => {
      // Refresh both namespaces (for stats) and requests
      queryClient.invalidateQueries({ queryKey: NAMESPACE_KEYS.all });
      queryClient.invalidateQueries({ queryKey: ['requests'] });
    },
  });
}
