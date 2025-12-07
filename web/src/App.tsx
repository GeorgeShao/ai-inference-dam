import { useState, useMemo } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { NamespaceTable } from '@/components/NamespaceTable';
import { RequestTable } from '@/components/RequestTable';
import { useNamespaces } from '@/hooks/useNamespaces';
import { useRequests } from '@/hooks/useRequests';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000,  // Consider data stale after 1 second
      retry: 1,
    },
  },
});

function Dashboard() {
  const { data: namespaces } = useNamespaces();
  const defaultTab = namespaces?.[0]?.name || 'default';
  const [selectedTab, setSelectedTab] = useState<string | undefined>(undefined);

  const activeTab = selectedTab || defaultTab;
  const { data: requestsData } = useRequests({ namespace: activeTab, limit: 1000 });

  const stats = useMemo(() => {
    if (!requestsData?.requests) return null;
    const counts = { queued: 0, processing: 0, completed: 0, failed: 0 };
    for (const req of requestsData.requests) {
      if (req.status in counts) {
        counts[req.status as keyof typeof counts]++;
      }
    }
    return { total: requestsData.total, ...counts };
  }, [requestsData]);

  return (
    <div className="container mx-auto py-6 px-4">
      <h1 className="text-2xl font-bold mb-6">AI Inference Dam</h1>

      <div className="mb-8">
        <NamespaceTable />
      </div>

      <div>
        <Tabs value={activeTab} onValueChange={setSelectedTab} className="w-full">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-4">
              <h2 className="text-lg font-semibold">Requests</h2>
              <TabsList>
                {namespaces?.map((ns) => (
                  <TabsTrigger key={ns.name} value={ns.name}>
                    {ns.name}
                  </TabsTrigger>
                ))}
              </TabsList>
            </div>
            {stats && (
              <div className="flex items-center gap-3 text-sm">
                <span className="text-muted-foreground">{stats.total} total</span>
                <span className="text-blue-600">{stats.queued} queued</span>
                <span className="text-yellow-600">{stats.processing} processing</span>
                <span className="text-green-600">{stats.completed} completed</span>
                <span className="text-red-600">{stats.failed} failed</span>
              </div>
            )}
          </div>

          {namespaces?.map((ns) => (
            <TabsContent key={ns.name} value={ns.name} className="mt-0">
              <RequestTable namespace={ns.name} />
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Dashboard />
    </QueryClientProvider>
  );
}
