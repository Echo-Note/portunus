import { Row, Col, Card, Statistic, Button, Spin, Typography, Empty } from 'antd';
import {
  ProjectOutlined,
  GlobalOutlined,
  WarningOutlined,
  TeamOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useState } from 'react';
import { useProjects } from '../../features/projects/hooks';
import { ProjectCard } from '../../features/projects/components/ProjectCard';
import { ProjectCreateModal } from '../../features/projects/components/ProjectCreateModal';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorResult } from '../../components/ui/ErrorResult';
import { PageHeader } from '../../components/ui/PageHeader';

const { Title } = Typography;

/**
 * 仪表盘页面：统计卡片 + 项目列表。
 */
export function DashboardPage() {
  const { data: projects = [], isLoading, isError, refetch } = useProjects();
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const totalDomains = projects.reduce((sum, p) => sum + p.max_domains, 0);
  const errorDomains = projects.filter((p) => p.status === 'error').length;
  const totalMembers = projects.reduce((sum, p) => sum + p.max_members, 0);

  return (
    <div>
      <PageHeader
        title="仪表盘"
        breadcrumbs={[{ title: '首页' }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            创建项目
          </Button>
        }
      />

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="项目总数" value={projects.length} prefix={<ProjectOutlined />} valueStyle={{ color: '#1677ff' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="活跃域名数" value={totalDomains} prefix={<GlobalOutlined />} valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="异常项目数" value={errorDomains} prefix={<WarningOutlined />} valueStyle={{ color: errorDomains > 0 ? '#ff4d4f' : '#52c41a' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="成员数" value={totalMembers} prefix={<TeamOutlined />} valueStyle={{ color: '#722ed1' }} />
          </Card>
        </Col>
      </Row>

      <Title level={4} style={{ marginBottom: 16 }}>项目列表</Title>

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>
      ) : isError ? (
        <ErrorResult onRetry={() => refetch()} />
      ) : projects.length === 0 ? (
        <EmptyState
          description="还没有项目，点击右上角按钮创建第一个项目"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            创建项目
          </Button>
        </EmptyState>
      ) : (
        <Row gutter={[16, 16]}>
          {projects.map((project) => (
            <Col xs={24} sm={12} lg={8} key={project.id}>
              <ProjectCard project={project} />
            </Col>
          ))}
        </Row>
      )}

      <ProjectCreateModal open={createModalOpen} onClose={() => setCreateModalOpen(false)} />
    </div>
  );
}