import { Tag } from 'antd';
import { STATUS_COLORS } from '../../types/rbac';

interface StatusTagProps {
  status: string;
  label?: string;
}

export function StatusTag({ status, label }: StatusTagProps) {
  const color = STATUS_COLORS[status] || 'default';
  return <Tag color={color}>{label ?? status}</Tag>;
}