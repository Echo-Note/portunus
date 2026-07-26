import { Row, Col, Spin, Button, Space, Popconfirm } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useState } from 'react';
import { useDomain } from '../../features/domains/hooks';
import { useDeleteDomain } from '../../features/domains/hooks';
import { useMembers } from '../../features/members/hooks';
import { useAuthStore } from '../../features/auth/store';
import { hasPermission } from '../../types/rbac';
import { StatusTag } from '../../components/ui/StatusTag';
import { PageHeader } from '../../components/ui/PageHeader';
import { ErrorResult } from '../../components/ui/ErrorResult';
import { ProxyConfigForm } from '../../features/proxy/components/ProxyConfigForm';
import { UpstreamTable } from '../../features/proxy/components/UpstreamTable';
import { UpstreamStatusChart } from '../../features/proxy/components/UpstreamStatusChart';
import { DomainEditModal } from '../../features/domains/components/DomainEditModal';

/**
 * 域名详情页：代理配置 + 上游列表 + 健康状态。
 */
export function DomainDetailPage() {
  const { projectId, domainId } = useParams<{ projectId: string; domainId: string }>();
  const navigate = useNavigate();
  const { data: domain, isLoading, isError, refetch } = useDomain(projectId!, domainId!);
  const { data: members = [] } = useMembers(projectId!);
  const deleteDomain = useDeleteDomain(projectId!);
  const currentUser = useAuthStore((s) => s.user);
  const [editModalOpen, setEditModalOpen] = useState(false);

  const currentMember = members.find((m) => m.user_id === currentUser?.id);
  const currentRole = currentMember?.role;
  const canEdit = hasPermission(currentRole, 'editDomain');
  const canDelete = hasPermission(currentRole, 'deleteDomain');

  if (isLoading) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  if (isError || !domain) {
    return <ErrorResult message="无法加载域名信息" onRetry={() => refetch()} />;
  }

  const handleDelete = async () => {
    await deleteDomain.mutateAsync(domainId!);
    navigate(`/projects/${projectId}`);
  };

  return (
    <div>
      <PageHeader
        title={
          <Space>
            {domain.domain_name}
            <StatusTag status={domain.status} />
            {domain.ssl_enabled && <StatusTag status="active" label="SSL" />}
          </Space>
        }
        breadcrumbs={[
          { title: '仪表盘', path: '/' },
          { title: '项目', path: `/projects/${projectId}` },
          { title: domain.domain_name },
        ]}
        extra={
          <Space>
            {canEdit && <Button icon={<EditOutlined />} onClick={() => setEditModalOpen(true)}>编辑</Button>}
            {canDelete && (
              <Popconfirm title={`确定要删除 ${domain.domain_name} 吗？此操作不可撤销。`} onConfirm={handleDelete}>
                <Button danger icon={<DeleteOutlined />} loading={deleteDomain.isPending}>删除</Button>
              </Popconfirm>
            )}
          </Space>
        }
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <UpstreamTable projectId={projectId!} domainId={domainId!} currentUserRole={currentRole} />
        </Col>
        <Col xs={24} lg={10}>
          <UpstreamStatusChart projectId={projectId!} domainId={domainId!} />
        </Col>
      </Row>

      <div style={{ marginTop: 16 }}>
        <ProxyConfigForm projectId={projectId!} domainId={domainId!} />
      </div>

      <DomainEditModal open={editModalOpen} projectId={projectId!} domain={domain} onClose={() => setEditModalOpen(false)} />
    </div>
  );
}