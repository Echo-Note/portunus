import { Layout, Spin } from 'antd';
import { Outlet, useNavigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from '../../features/auth/store';
import { Sidebar } from './Sidebar';
import { UserMenu } from './UserMenu';
import { authApi } from '../../features/auth/api';

const { Header, Sider, Content } = Layout;

/**
 * 应用主布局：左侧边栏 + 顶部导航 + 内容区。
 */
export function AppLayout() {
  const { user, setUser, logout, accessToken } = useAuthStore();
  const navigate = useNavigate();

  useEffect(() => {
    if (accessToken && !user) {
      authApi.getMe()
        .then((u) => setUser(u))
        .catch(() => {
          logout();
          navigate('/login');
        });
    }
  }, [accessToken, user, setUser, logout, navigate]);

  if (!user) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        width={220}
        style={{
          background: '#001529',
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: 20,
            fontWeight: 700,
            borderBottom: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          Portunus
        </div>
        <Sidebar />
      </Sider>
      <Layout style={{ marginLeft: 220 }}>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
            borderBottom: '1px solid #f0f0f0',
            position: 'sticky',
            top: 0,
            zIndex: 10,
          }}
        >
          <UserMenu />
        </Header>
        <Content style={{ padding: 24, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}