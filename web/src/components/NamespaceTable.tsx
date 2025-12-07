import { useState, useMemo, useCallback } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef, ICellRendererParams } from 'ag-grid-community';
import { AllCommunityModule, ModuleRegistry } from 'ag-grid-community';

import { Button } from '@/components/ui/button';
import { NamespaceDialog } from './NamespaceDialog';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';
import { useNamespaces, useDispatch } from '@/hooks/useNamespaces';
import type { Namespace } from '@/types';
import { Plus, Pencil, Trash2, Play } from 'lucide-react';

ModuleRegistry.registerModules([AllCommunityModule]);

export function NamespaceTable() {
  const { data: namespaces, isLoading, error } = useNamespaces();
  const dispatchMutation = useDispatch();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedNamespace, setSelectedNamespace] = useState<Namespace | null>(null);

  const handleEdit = useCallback((namespace: Namespace) => {
    setSelectedNamespace(namespace);
    setEditDialogOpen(true);
  }, []);

  const handleDelete = useCallback((namespace: Namespace) => {
    setSelectedNamespace(namespace);
    setDeleteDialogOpen(true);
  }, []);

  const handleDispatch = useCallback(async (namespace: Namespace) => {
    try {
      await dispatchMutation.mutateAsync(namespace.name);
    } catch (error) {
      console.error('Failed to dispatch:', error);
    }
  }, [dispatchMutation]);

  // AG Grid column definitions
  const columnDefs = useMemo<ColDef<Namespace>[]>(() => [
    {
      field: 'name',
      headerName: 'Name',
      flex: 1,
      minWidth: 120,
      sortable: true,
      filter: true,
    },
    {
      field: 'description',
      headerName: 'Description',
      flex: 2,
      minWidth: 150,
    },
    {
      headerName: 'Provider Endpoint',
      valueGetter: (params) => params.data?.provider?.api_endpoint || '-',
      flex: 2,
      minWidth: 200,
    },
    {
      headerName: 'Model',
      valueGetter: (params) => params.data?.provider?.model || '-',
      flex: 1,
      minWidth: 100,
    },
    {
      headerName: 'Stats',
      cellRenderer: (params: ICellRendererParams<Namespace>) => {
        const stats = params.data?.stats;
        if (!stats) return '-';
        return (
          <div className="flex gap-2 text-xs">
            <span className="text-blue-600">{stats.queued} Q</span>
            <span className="text-yellow-600">{stats.processing} P</span>
            <span className="text-green-600">{stats.completed} C</span>
            <span className="text-red-600">{stats.failed} F</span>
          </div>
        );
      },
      flex: 2,
      minWidth: 180,
    },
    {
      field: 'created_at',
      headerName: 'Created At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
      sortable: true,
    },
    {
      field: 'updated_at',
      headerName: 'Updated At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
      sortable: true,
    },
    {
      headerName: 'Actions',
      cellRenderer: (params: ICellRendererParams<Namespace>) => {
        const namespace = params.data;
        if (!namespace) return null;

        const isDefault = namespace.name === 'default';

        return (
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleDispatch(namespace)}
              title="Dispatch"
              disabled={dispatchMutation.isPending}
            >
              <Play className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleEdit(namespace)}
              title="Edit"
            >
              <Pencil className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleDelete(namespace)}
              title="Delete"
              disabled={isDefault}
              className={isDefault ? 'opacity-50 cursor-not-allowed' : ''}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        );
      },
      flex: 1,
      minWidth: 130,
      sortable: false,
      filter: false,
    },
  ], [handleEdit, handleDelete, handleDispatch, dispatchMutation.isPending]);

  const defaultColDef = useMemo<ColDef>(() => ({
    resizable: true,
  }), []);

  if (error) {
    return <div className="text-red-500">Error loading namespaces: {error.message}</div>;
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold">Namespaces</h2>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create Namespace
        </Button>
      </div>

      <div className="flex-1" style={{ width: '100%', minHeight: 400 }}>
        <AgGridReact<Namespace>
          rowData={namespaces || []}
          columnDefs={columnDefs}
          defaultColDef={defaultColDef}
          loading={isLoading}
          domLayout="autoHeight"
          animateRows
          rowSelection="single"
        />
      </div>

      <NamespaceDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />

      <NamespaceDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        namespace={selectedNamespace || undefined}
      />

      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        namespaceName={selectedNamespace?.name || ''}
      />
    </div>
  );
}
