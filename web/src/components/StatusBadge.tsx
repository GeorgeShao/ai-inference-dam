import { Badge } from '@/components/ui/badge';
import type { RequestStatus } from '@/types';

interface StatusBadgeProps {
  status: RequestStatus;
}

const statusConfig: Record<RequestStatus, { label: string; className: string }> = {
  queued: {
    label: 'Queued',
    className: 'bg-blue-100 text-blue-800 hover:bg-blue-100',
  },
  processing: {
    label: 'Processing',
    className: 'bg-yellow-100 text-yellow-800 hover:bg-yellow-100',
  },
  completed: {
    label: 'Completed',
    className: 'bg-green-100 text-green-800 hover:bg-green-100',
  },
  failed: {
    label: 'Failed',
    className: 'bg-red-100 text-red-800 hover:bg-red-100',
  },
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <Badge variant="outline" className={config.className}>
      {config.label}
    </Badge>
  );
}
