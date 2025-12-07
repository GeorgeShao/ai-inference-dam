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

  return (
    <div className="container mx-auto py-6 px-4">
      <h1 className="text-2xl font-bold mb-6">AI Inference Dam</h1>

      <Tabs defaultValue="namespaces" className="w-full">
        <TabsList className="mb-4">
          <TabsTrigger value="namespaces">Namespaces</TabsTrigger>
          {namespaces?.map((ns) => (
            <TabsTrigger key={ns.name} value={`requests-${ns.name}`}>
              {ns.name}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="namespaces" className="mt-0">
          <NamespaceTable />
        </TabsContent>

        {namespaces?.map((ns) => (
          <TabsContent key={ns.name} value={`requests-${ns.name}`} className="mt-0">
            <RequestTable namespace={ns.name} />
          </TabsContent>
        ))}
      </Tabs>
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
