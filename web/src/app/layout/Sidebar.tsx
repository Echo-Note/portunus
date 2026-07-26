import { Menu, Spin } from 'antd';
import {
  DashboardOutlined,
  ProjectOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, useParams } from 'react-router-dom';
import { useState } from 'react';
import { useProjects } from '../../features/projects/hooks';
import { ProjectCreateModal } from '../../features/projects/components/ProjectCreateModal';

/**
 * 侧边导航：项目列表 + 创建项目按钮。
 */
export function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const { projectId } = useParams();
  const { data: projects = [], isLoading } = useProjects();
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const selectedKeys: string[] = [];
  if (location.pathname === '/') {
    selectedKeys.push('dashboard');
  } else if (projectId) {
    selectedKeys.push(projectId);
  }

  const menuItems = [
    {
      key: 'dashboard',
      icon: <DashboardOutlined />,
      label: '仪表盘',
      onClick: () => navigate('/'),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'projects-header',
      label: '项目',
      type: 'group' as const,
      children: isLoading
        ? [{
            key: 'loading',
            label: <Spin size="small" />,
            disabled: true,
          }]
        : [
            ...(projects.map((p) => ({
              key: p.id,
              icon: <ProjectOutlined />,
              label: p.name,
              onClick: () => navigate(`/projects/${p.id}`),
            }))),
            {
              key: 'create-project',
              icon: <PlusOutlined />,
              label: '创建项目',
              onClick: () => setCreateModalOpen(true),
            },
          ],
    },
  ];

  return (
    <>
      <Menu
        mode="inline"
        selectedKeys={selectedKeys}
        style={{ borderRight: 0, background: 'transparent', color: '#fff' }}
        theme="dark"
        items={menuItems}
      />
      <ProjectCreateModal open={createModalOpen} onClose={() => setCreateModalOpen(false)} />
    </>
  );
}