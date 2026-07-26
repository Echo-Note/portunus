import { Button, Popconfirm, Tooltip } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useTokens, useRevokeToken } from '../hooks';
import type { ApiToken } from '../types';
import { DataTable } from '../../../components/ui/DataTable';
import { TokenCreateModal } from './TokenCreateModal';
import type { ColumnsType } from 'antd/es/table';

export function TokenTable() {
  const { data: tokens = [], isLoading, isError, refetch } = useTokens();
  const revokeToken = useRevokeToken();
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const columns: ColumnsType<ApiToken> = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '操作', key: 'actions', width: 100,
      render: (_: unknown, record: ApiToken) => (
        <Popconfirm title="确定要撤销此 Token 吗？" onConfirm={() => revokeToken.mutate(record.id)}>
          <Tooltip title="撤销"><Button type="link" size="small" danger icon={<DeleteOutlined />} loading={revokeToken.isPending} /></Tooltip>
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <DataTable
        columns={columns}
        dataSource={tokens}
        rowKey="id"
        loading={isLoading}
        error={isError}
        onRetry={() => refetch()}
        emptyText="暂无 Token"
        pagination={false}
        size="small"
        toolbar={
          <>
            <strong>API Token</strong>
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={() => setCreateModalOpen(true)}>创建 Token</Button>
          </>
        }
      />
      <TokenCreateModal open={createModalOpen} onClose={() => setCreateModalOpen(false)} />
    </>
  );
}