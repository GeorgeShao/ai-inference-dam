import { useMemo } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef, ICellRendererParams } from 'ag-grid-community';
import { AllCommunityModule, ModuleRegistry } from 'ag-grid-community';

import { StatusBadge } from './StatusBadge';
import { useRequests } from '@/hooks/useRequests';
import type { Request, RequestStatus } from '@/types';

ModuleRegistry.registerModules([AllCommunityModule]);

interface RequestTableProps {
  namespace: string;
}

export function RequestTable({ namespace }: RequestTableProps) {
  const { data, isLoading, error } = useRequests({ namespace, limit: 1000 });

  const columnDefs = useMemo<ColDef<Request>[]>(() => [
    {
      field: 'id',
      headerName: 'ID',
      flex: 2,
      minWidth: 250,
      sortable: true,
      filter: true,
    },
    {
      field: 'namespace',
      headerName: 'Namespace',
      flex: 1,
      minWidth: 120,
      sortable: true,
      filter: true,
    },
    {
      field: 'status',
      headerName: 'Status',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        const status = params.value as RequestStatus;
        return <StatusBadge status={status} />;
      },
      flex: 1,
      minWidth: 100,
      sortable: true,
      filter: true,
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
      field: 'dispatched_at',
      headerName: 'Dispatched At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
      sortable: true,
    },
    {
      field: 'completed_at',
      headerName: 'Completed At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
      sortable: true,
    },
    {
      field: 'error',
      headerName: 'Error',
      flex: 2,
      minWidth: 200,
      cellStyle: { color: 'red' },
      valueFormatter: (params) => params.value || '-',
    },
  ], []);

  const defaultColDef = useMemo<ColDef>(() => ({
    resizable: true,
  }), []);

  if (error) {
    return <div className="text-red-500">Error loading requests: {error.message}</div>;
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold">Requests - {namespace}</h2>
        <div className="text-sm text-muted-foreground">
          {data?.total !== undefined && `Total: ${data.total} requests`}
        </div>
      </div>

      <div className="flex-1" style={{ width: '100%', minHeight: 400 }}>
        <AgGridReact<Request>
          rowData={data?.requests || []}
          columnDefs={columnDefs}
          defaultColDef={defaultColDef}
          loading={isLoading}
          domLayout="autoHeight"
          animateRows
          rowSelection="single"
          getRowId={(params) => params.data.id}
        />
      </div>
    </div>
  );
}
