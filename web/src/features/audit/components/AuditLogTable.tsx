import { Table, Tag } from 'antd';
import { useAuditLogs } from '../hooks';
import type { AuditLog } from '../types';
import { StatusTag } from '../../../components/ui/StatusTag';
import { DataTable } from '../../../components/ui/DataTable';
import type { ColumnsType } from 'antd/es/table';

interface AuditLogTableProps { projectId: string; }

const VIA_LABELS: Record<string, string> = { web_ui: 'Web UI', mcp_tool: 'MCP', api_token: 'API Token', system: '系统' };

export function AuditLogTable({ projectId }: AuditLogTableProps) {
  const { data, isLoading, isError, refetch } = useAuditLogs(projectId);
  const logs = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns: ColumnsType<AuditLog> = [
    { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
    { title: '资源类型', dataIndex: 'resource_type', key: 'resource_type', width: 120 },
    { title: '执行者', dataIndex: 'actor_name', key: 'actor_name', width: 200 },
    { title: '执行者类型', dataIndex: 'actor_type', key: 'actor_type', width: 100, render: (t: string) => <Tag>{t}</Tag> },
    { title: '来源', dataIndex: 'via', key: 'via', width: 100, render: (via: string) => <Tag>{VIA_LABELS[via] || via}</Tag> },
    { title: '结果', dataIndex: 'result', key: 'result', width: 80, render: (r: string) => <StatusTag status={r} /> },
  ];

  return (
    <DataTable
      columns={columns}
      dataSource={logs}
      rowKey="id"
      loading={isLoading}
      error={isError}
      onRetry={() => refetch()}
      emptyText="暂无审计日志"
      pagination={{ total, showTotal: (t) => `共 ${t} 条` }}
    />
  );
}