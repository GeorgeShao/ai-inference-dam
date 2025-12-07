import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { NamespaceTable } from '@/components/NamespaceTable';
import { RequestTable } from '@/components/RequestTable';
import { useNamespaces } from '@/hooks/useNamespaces';

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

  return (
    <div className="container mx-auto py-6 px-4">
      <h1 className="text-2xl font-bold mb-6">AI Inference Dam</h1>

      <div className="mb-8">
        <NamespaceTable />
      </div>

      <div>
        <Tabs value={activeTab} onValueChange={setSelectedTab} className="w-full">
          <div className="flex items-center gap-4 mb-4">
            <h2 className="text-lg font-semibold">Requests</h2>
            <TabsList>
              {namespaces?.map((ns) => (
                <TabsTrigger key={ns.name} value={ns.name}>
                  {ns.name}
                </TabsTrigger>
              ))}
            </TabsList>
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
