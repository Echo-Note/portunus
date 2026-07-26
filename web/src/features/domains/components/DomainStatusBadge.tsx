import { Tag } from 'antd';
import { STATUS_COLORS } from '../../../types/rbac';

interface DomainStatusBadgeProps {
  status: string;
}

/**
 * 域名状态徽章。
 */
export function DomainStatusBadge({ status }: DomainStatusBadgeProps) {
  const color = STATUS_COLORS[status] || 'default';
  return <Tag color={color}>{status}</Tag>;
}
