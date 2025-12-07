import { useMemo, useState, useCallback, useRef, useEffect } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef, ICellRendererParams, IDatasource, IGetRowsParams } from 'ag-grid-community';
import { AllCommunityModule, ModuleRegistry } from 'ag-grid-community';
import { Copy, Check } from 'lucide-react';

import { StatusBadge } from './StatusBadge';
import { ContentDialog } from './ContentDialog';
import * as api from '@/api/client';
import type { Request, RequestStatus, NamespaceStats } from '@/types';

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

const BLOCK_SIZE = 100;

interface RequestTableProps {
  namespace: string;
  stats?: NamespaceStats;
}

export function RequestTable({ namespace }: RequestTableProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogTitle, setDialogTitle] = useState('');
  const [dialogContent, setDialogContent] = useState('');
  const gridRef = useRef<AgGridReact<Request>>(null);

  // Track cursor state for pagination
  const cursorStateRef = useRef<{
    // Map of startRow -> cursor to use for that block
    cursors: Map<number, string | null>;
    // Total count from the API
    totalCount: number | null;
    // Whether we've reached the last row
    reachedEnd: boolean;
  }>({
    cursors: new Map([[0, null]]), // First block has no cursor
    totalCount: null,
    reachedEnd: false,
  });

  const openContentDialog = useCallback((title: string, content: string) => {
    setDialogTitle(title);
    setDialogContent(content);
    setDialogOpen(true);
  }, []);

  // Create datasource for infinite scrolling
  const datasource = useMemo<IDatasource>(() => ({
    getRows: async (params: IGetRowsParams) => {
      const { startRow, successCallback, failCallback } = params;
      const state = cursorStateRef.current;

      try {
        // Find the cursor for this block
        // For cursor pagination, we need to load blocks sequentially
        // Check if we have a cursor for this startRow
        let cursor = state.cursors.get(startRow);

        // If we don't have a cursor and this isn't the first block,
        // we need to load preceding blocks first
        if (cursor === undefined && startRow > 0) {
          // Find the closest loaded block before this one
          const sortedStarts = Array.from(state.cursors.keys()).sort((a, b) => a - b);
          const closestStart = sortedStarts.filter(s => s < startRow).pop();

          if (closestStart !== undefined) {
            // Load all blocks between closestStart and startRow
            let currentStart = closestStart;
            while (currentStart < startRow) {
              const prevCursor = state.cursors.get(currentStart) ?? null;
              const response = await api.listRequests({
                namespace,
                limit: BLOCK_SIZE,
                cursor: prevCursor ?? undefined,
              });

              state.totalCount = response.total;
              const nextStart = currentStart + BLOCK_SIZE;

              if (response.next_cursor) {
                state.cursors.set(nextStart, response.next_cursor);
              } else {
                state.reachedEnd = true;
              }

              currentStart = nextStart;
            }
            cursor = state.cursors.get(startRow);
          }
        }

        // Fetch the actual data for this block
        const response = await api.listRequests({
          namespace,
          limit: BLOCK_SIZE,
          cursor: cursor ?? undefined,
        });

        state.totalCount = response.total;

        // Store cursor for next block
        const nextStartRow = startRow + response.requests.length;
        if (response.next_cursor) {
          state.cursors.set(nextStartRow, response.next_cursor);
        } else {
          state.reachedEnd = true;
        }

        // Determine lastRow for AG Grid
        // If we've reached the end, tell AG Grid the exact last row
        // Otherwise, return -1 to indicate more data may be available
        let lastRow = -1;
        if (state.reachedEnd || !response.next_cursor) {
          lastRow = startRow + response.requests.length;
        } else if (state.totalCount !== null) {
          // We know the total, use it
          lastRow = state.totalCount;
        }

        successCallback(response.requests, lastRow);
      } catch (error) {
        console.error('Failed to load requests:', error);
        failCallback();
      }
    },
  }), [namespace]);

  // Reset datasource when namespace changes
  useEffect(() => {
    cursorStateRef.current = {
      cursors: new Map([[0, null]]),
      totalCount: null,
      reachedEnd: false,
    };

    // Refresh the grid data
    if (gridRef.current?.api) {
      gridRef.current.api.setGridOption('datasource', datasource);
    }
  }, [namespace, datasource]);

  const columnDefs = useMemo<ColDef<Request>[]>(() => [
    {
      field: 'id',
      headerName: 'ID',
      flex: 2,
      minWidth: 250,
    },
    {
      field: 'status',
      headerName: 'Status',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.data) return null;
        const status = params.value as RequestStatus;
        return <StatusBadge status={status} />;
      },
      flex: 1,
      minWidth: 100,
    },
    {
      field: 'request',
      headerName: 'Request',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.data) return null;
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
        if (!params.data) return null;
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
    },
    {
      field: 'dispatched_at',
      headerName: 'Dispatched At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
    },
    {
      field: 'completed_at',
      headerName: 'Completed At',
      valueFormatter: (params) =>
        params.value ? new Date(params.value).toLocaleString() : '-',
      flex: 1,
      minWidth: 150,
    },
    {
      field: 'error',
      headerName: 'Error',
      cellRenderer: (params: ICellRendererParams<Request>) => {
        if (!params.data) return null;
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
    sortable: false,  // Disable sorting globally (not compatible with cursor pagination)
  }), []);

  return (
    <div>
      <div style={{ width: '100%', height: '600px' }}>
        <AgGridReact<Request>
          ref={gridRef}
          columnDefs={columnDefs}
          defaultColDef={defaultColDef}
          rowModelType="infinite"
          datasource={datasource}
          cacheBlockSize={BLOCK_SIZE}
          cacheOverflowSize={2}
          maxConcurrentDatasourceRequests={1}
          infiniteInitialRowCount={BLOCK_SIZE}
          maxBlocksInCache={100}
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