import { Badge } from '@/components/ui/badge';
import { AlertCircle, AlertTriangle, Info } from 'lucide-react';

export type AlertSeverity = 'info' | 'warning' | 'critical';

interface AlertBadgeProps {
  severity: AlertSeverity;
  count?: number;
  showIcon?: boolean;
}

export function AlertBadge({ severity, count, showIcon = true }: AlertBadgeProps) {
  const getIcon = () => {
    switch (severity) {
      case 'critical':
        return <AlertCircle className="h-3 w-3" />;
      case 'warning':
        return <AlertTriangle className="h-3 w-3" />;
      case 'info':
        return <Info className="h-3 w-3" />;
    }
  };

  const getVariant = () => {
    switch (severity) {
      case 'critical':
        return 'destructive';
      case 'warning':
        return 'warning' as any; // Custom variant
      case 'info':
        return 'secondary';
    }
  };

  const getText = () => {
    const labels = {
      critical: '严重',
      warning: '警告',
      info: '信息',
    };
    return count ? `${labels[severity]} (${count})` : labels[severity];
  };

  return (
    <Badge variant={getVariant()} className="gap-1">
      {showIcon && getIcon()}
      <span>{getText()}</span>
    </Badge>
  );
}
