import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useDeleteNamespace } from '@/hooks/useNamespaces';

interface DeleteConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  namespaceName: string;
}

export function DeleteConfirmDialog({ open, onOpenChange, namespaceName }: DeleteConfirmDialogProps) {
  const deleteMutation = useDeleteNamespace();

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(namespaceName);
      onOpenChange(false);
    } catch (error) {
      console.error('Failed to delete namespace:', error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Namespace</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete the namespace "{namespaceName}"?
            This action cannot be undone and will delete all associated requests.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
