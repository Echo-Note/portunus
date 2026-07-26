import { Empty, EmptyProps } from 'antd';

interface EmptyStateProps extends EmptyProps {
  description?: string;
}

export function EmptyState({ description = '暂无数据', ...rest }: EmptyStateProps) {
  return <Empty description={description} {...rest} />;
}