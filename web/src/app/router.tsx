import { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';
import { Spin } from 'antd';
import { AuthGuard } from '../features/auth/components/AuthGuard';
import { AppLayout } from './layout/AppLayout';

// 懒加载页面
const LoginPage = lazy(() => import('./routes/LoginPage').then((m) => ({ default: m.LoginPage })));
const RegisterPage = lazy(() => import('./routes/RegisterPage').then((m) => ({ default: m.RegisterPage })));
const DashboardPage = lazy(() => import('./routes/DashboardPage').then((m) => ({ default: m.DashboardPage })));
const ProjectDetailPage = lazy(() => import('./routes/ProjectDetailPage').then((m) => ({ default: m.ProjectDetailPage })));
const DomainDetailPage = lazy(() => import('./routes/DomainDetailPage').then((m) => ({ default: m.DomainDetailPage })));
const InvitationPage = lazy(() => import('./routes/InvitationPage').then((m) => ({ default: m.InvitationPage })));
const SettingsPage = lazy(() => import('./routes/SettingsPage').then((m) => ({ default: m.SettingsPage })));
const NotFoundPage = lazy(() => import('./routes/NotFoundPage').then((m) => ({ default: m.NotFoundPage })));

function PageLoading() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      <Spin size="large" />
    </div>
  );
}

/**
 * 路由配置：公开路由 + 受保护路由。
 */
export function Router() {
  return (
    <Suspense fallback={<PageLoading />}>
      <Routes>
        {/* 公开路由 */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />

        {/* 邀请路由 */}
        <Route
          path="/invitations/:token"
          element={
            <AuthGuard>
              <InvitationPage />
            </AuthGuard>
          }
        />

        {/* 受保护路由 */}
        <Route
          element={
            <AuthGuard>
              <AppLayout />
            </AuthGuard>
          }
        >
          <Route path="/" element={<DashboardPage />} />
          <Route path="/projects/:projectId" element={<ProjectDetailPage />} />
          <Route path="/projects/:projectId/domains/:domainId" element={<DomainDetailPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  );
}