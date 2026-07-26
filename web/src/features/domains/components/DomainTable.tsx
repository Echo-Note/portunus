import { Button, Space, Popconfirm, Tag, Tooltip } from 'antd';
import { EditOutlined, DeleteOutlined, LinkOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useDomains, useDeleteDomain } from '../hooks';
import type { Domain } from '../types';
import { hasPermission } from '../../../types/rbac';
import { StatusTag } from '../../../components/ui/StatusTag';
import { DataTable } from '../../../components/ui/DataTable';
import type { ColumnsType } from 'antd/es/table';

interface DomainTableProps {
  projectId: string;
  currentUserRole?: string;
}

export function DomainTable({ projectId, currentUserRole }: DomainTableProps) {
  const navigate = useNavigate();
  const { data: domains = [], isLoading, isError, refetch } = useDomains(projectId);
  const deleteDomain = useDeleteDomain(projectId);
  const canEdit = hasPermission(currentUserRole, 'editDomain');
  const canDelete = hasPermission(currentUserRole, 'deleteDomain');

  const columns: ColumnsType<Domain> = [
    {
      title: '域名', dataIndex: 'domain_name', key: 'domain_name',
      render: (name: string, record: Domain) => (
        <a onClick={() => navigate(`/projects/${projectId}/domains/${record.id}`)}>
          <LinkOutlined style={{ marginRight: 8 }} />{name}
        </a>
      ),
    },
    { title: '状态', dataIndex: 'status', key: 'status', width: 120, render: (s: string) => <StatusTag status={s} /> },
    { title: 'SSL', dataIndex: 'ssl_enabled', key: 'ssl_enabled', width: 80, render: (v: boolean) => v ? <Tag color="green">已启用</Tag> : <Tag>未启用</Tag> },
    {
      title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: Domain) => (
        <Space>
          {canEdit && (
            <Tooltip title="编辑"><Button type="link" size="small" icon={<EditOutlined />}
              onClick={() => navigate(`/projects/${projectId}/domains/${record.id}`)} /></Tooltip>
          )}
          {canDelete && (
            <Popconfirm title={`确定要删除 ${record.domain_name} 吗？此操作不可撤销。`} onConfirm={() => deleteDomain.mutate(record.id)}>
              <Tooltip title="删除"><Button type="link" size="small" danger icon={<DeleteOutlined />} /></Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      dataSource={domains}
      rowKey="id"
      loading={isLoading}
      error={isError}
      onRetry={() => refetch()}
      emptyText="暂无域名"
    />
  );
}