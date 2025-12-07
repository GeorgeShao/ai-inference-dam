import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useCreateNamespace, useUpdateNamespace } from '@/hooks/useNamespaces';
import type { Namespace, CreateNamespaceRequest, UpdateNamespaceRequest } from '@/types';

interface NamespaceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  namespace?: Namespace;  // If provided, edit mode; otherwise create mode
}

export function NamespaceDialog({ open, onOpenChange, namespace }: NamespaceDialogProps) {
  const isEditMode = !!namespace;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [apiEndpoint, setApiEndpoint] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('');

  const createMutation = useCreateNamespace();
  const updateMutation = useUpdateNamespace();

  const isLoading = createMutation.isPending || updateMutation.isPending;

  // Reset form when dialog opens/closes or namespace changes
  useEffect(() => {
    if (open && namespace) {
      setName(namespace.name);
      setDescription(namespace.description || '');
      setApiEndpoint(namespace.provider?.api_endpoint || '');
      setApiKey('');  // Never pre-fill API key (security)
      setModel(namespace.provider?.model || '');
    } else if (open && !namespace) {
      setName('');
      setDescription('');
      setApiEndpoint('');
      setApiKey('');
      setModel('');
    }
  }, [open, namespace]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const provider = {
      ...(apiEndpoint && { api_endpoint: apiEndpoint }),
      ...(apiKey && { api_key: apiKey }),
      ...(model && { model }),
    };

    const hasProvider = Object.keys(provider).length > 0;

    try {
      if (isEditMode) {
        const data: UpdateNamespaceRequest = {
          description,
          ...(hasProvider && { provider }),
        };
        await updateMutation.mutateAsync({ name: namespace.name, data });
      } else {
        const data: CreateNamespaceRequest = {
          name,
          description,
          ...(hasProvider && { provider }),
        };
        await createMutation.mutateAsync(data);
      }
      onOpenChange(false);
    } catch (error) {
      // Error handled by React Query
      console.error('Failed to save namespace:', error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditMode ? 'Edit Namespace' : 'Create Namespace'}</DialogTitle>
            <DialogDescription>
              {isEditMode
                ? 'Update the namespace configuration.'
                : 'Create a new namespace for organizing inference requests.'}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isEditMode}
                placeholder="my-namespace"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional description..."
                rows={2}
              />
            </div>

            <div className="border-t pt-4">
              <h4 className="text-sm font-medium mb-3">Provider Configuration</h4>

              <div className="grid gap-3">
                <div className="grid gap-2">
                  <Label htmlFor="apiEndpoint">API Endpoint</Label>
                  <Input
                    id="apiEndpoint"
                    value={apiEndpoint}
                    onChange={(e) => setApiEndpoint(e.target.value)}
                    placeholder="https://api.openai.com/v1"
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="apiKey">
                    API Key {isEditMode && <span className="text-muted-foreground">(leave blank to keep existing)</span>}
                  </Label>
                  <Input
                    id="apiKey"
                    type="password"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder="sk-..."
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="model">Model</Label>
                  <Input
                    id="model"
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder="gpt-4"
                  />
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
