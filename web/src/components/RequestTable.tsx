import { useMemo, useState, useCallback } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef, ICellRendererParams } from 'ag-grid-community';
import { AllCommunityModule, ModuleRegistry } from 'ag-grid-community';
import { Copy, Check } from 'lucide-react';

import { StatusBadge } from './StatusBadge';
import { ContentDialog } from './ContentDialog';
import { useRequests } from '@/hooks/useRequests';
import type { Request, RequestStatus } from '@/types';

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <button
      onClick={handleCopy}
      className="p-1 hover:bg-muted rounded shrink-0"
      title="Copy to clipboard"
    >
      {copied ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3 text-muted-foreground" />}
    </button>
  );
}

ModuleRegistry.registerModules([AllCommunityModule]);

interface RequestTableProps {
  namespace: string;
}

export function RequestTable({ namespace }: RequestTableProps) {
  const { data, isLoading, error } = useRequests({ namespace, limit: 1000 });
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogTitle, setDialogTitle] = useState('');
  const [dialogContent, setDialogContent] = useState('');

  const openContentDialog = useCallback((title: string, content: string) => {
    setDialogTitle(title);
    setDialogContent(content);
    setDialogOpen(true);
  }, []);

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
      field: 'request',
      headerName: 'Request',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.value) return <span className="text-muted-foreground">-</span>;
        const json = JSON.stringify(params.value, null, 2);
        // Extract user message content from request
        let preview = json;
        try {
          const messages = params.value.messages;
          if (Array.isArray(messages)) {
            const userMsg = messages.find((m: { role?: string }) => m.role === 'user');
            if (userMsg?.content) {
              preview = String(userMsg.content);
            }
          }
        } catch { /* fallback to json */ }
        const truncated = preview.length > 80 ? preview.slice(0, 80) + '...' : preview;
        return (
          <div className="flex items-center gap-1 w-full">
            <button
              onClick={() => openContentDialog('Request', json)}
              className="text-left hover:underline cursor-pointer truncate flex-1"
              title="Click to view full request"
            >
              {truncated}
            </button>
            <CopyButton text={json} />
          </div>
        );
      },
      flex: 2,
      minWidth: 200,
    },
    {
      field: 'response',
      headerName: 'Response',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.value) return <span className="text-muted-foreground">-</span>;
        const json = JSON.stringify(params.value, null, 2);
        // Extract assistant message content from response
        let preview = json;
        try {
          const choices = params.value.choices;
          if (Array.isArray(choices) && choices[0]?.message?.content) {
            preview = String(choices[0].message.content);
          }
        } catch { /* fallback to json */ }
        const truncated = preview.length > 80 ? preview.slice(0, 80) + '...' : preview;
        return (
          <div className="flex items-center gap-1 w-full">
            <button
              onClick={() => openContentDialog('Response', json)}
              className="text-left hover:underline cursor-pointer truncate flex-1"
              title="Click to view full response"
            >
              {truncated}
            </button>
            <CopyButton text={json} />
          </div>
        );
      },
      flex: 2,
      minWidth: 200,
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
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.value) return <span className="text-muted-foreground">-</span>;
        const errorText = String(params.value);
        const truncated = errorText.length > 60 ? errorText.slice(0, 60) + '...' : errorText;
        return (
          <div className="flex items-center gap-1 w-full">
            <button
              onClick={() => openContentDialog('Error', errorText)}
              className="text-left text-red-600 hover:underline cursor-pointer truncate flex-1"
              title="Click to view full error"
            >
              {truncated}
            </button>
            <CopyButton text={errorText} />
          </div>
        );
      },
      flex: 2,
      minWidth: 200,
    },
  ], [openContentDialog]);

  const defaultColDef = useMemo<ColDef>(() => ({
    resizable: true,
  }), []);

  if (error) {
    return <div className="text-red-500">Error loading requests: {error.message}</div>;
  }

  return (
    <div>
      <div style={{ width: '100%' }}>
        <AgGridReact<Request>
          rowData={data?.requests || []}
          columnDefs={columnDefs}
          defaultColDef={defaultColDef}
          loading={isLoading}
          domLayout="autoHeight"
          rowSelection="single"
          getRowId={(params) => params.data.id}
        />
      </div>

      <ContentDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title={dialogTitle}
        content={dialogContent}
      />
    </div>
  );
}
