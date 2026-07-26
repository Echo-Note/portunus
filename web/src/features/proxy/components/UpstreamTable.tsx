import { Button, Popconfirm, Tooltip } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useProxyConfig, useRemoveUpstream } from '../hooks';
import type { Upstream } from '../types';
import { hasPermission } from '../../../types/rbac';
import { StatusTag } from '../../../components/ui/StatusTag';
import { DataTable } from '../../../components/ui/DataTable';
import { UpstreamAddModal } from './UpstreamAddModal';
import type { ColumnsType } from 'antd/es/table';

interface UpstreamTableProps {
  projectId: string;
  domainId: string;
  currentUserRole?: string;
}

export function UpstreamTable({ projectId, domainId, currentUserRole }: UpstreamTableProps) {
  const { data: proxyConfig, isLoading, isError, refetch } = useProxyConfig(projectId, domainId);
  const removeUpstream = useRemoveUpstream(projectId, domainId);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const canManage = hasPermission(currentUserRole, 'manageUpstream');

  const upstreams: Upstream[] = (proxyConfig as unknown as { upstreams?: Upstream[] })?.upstreams ?? [];

  const columns: ColumnsType<Upstream> = [
    { title: '地址', dataIndex: 'dial_address', key: 'dial_address' },
    { title: '权重', dataIndex: 'weight', key: 'weight', width: 100, render: (w: number) => w || 1 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 120, render: (s: string) => <StatusTag status={s} /> },
    ...(canManage ? [{
      title: '操作', key: 'actions', width: 100,
      render: (_: unknown, record: Upstream) => (
        <Popconfirm title={`确定要移除上游 ${record.dial_address} 吗？此操作不可撤销。`} onConfirm={() => removeUpstream.mutate(record.id)}>
          <Tooltip title="移除"><Button type="link" size="small" danger icon={<DeleteOutlined />} loading={removeUpstream.isPending} /></Tooltip>
        </Popconfirm>
      ),
    }] : []),
  ];

  return (
    <>
      <DataTable
        columns={columns}
        dataSource={upstreams}
        rowKey="id"
        loading={isLoading}
        error={isError}
        onRetry={() => refetch()}
        emptyText="暂无上游"
        pagination={false}
        size="small"
        toolbar={
          <>
            <strong>上游列表</strong>
            {canManage && <Button type="primary" icon={<PlusOutlined />} size="small" onClick={() => setAddModalOpen(true)}>添加上游</Button>}
          </>
        }
      />
      <UpstreamAddModal open={addModalOpen} projectId={projectId} domainId={domainId} onClose={() => setAddModalOpen(false)} />
    </>
  );
}