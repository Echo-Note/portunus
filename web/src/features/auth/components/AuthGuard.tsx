import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store';

interface AuthGuardProps {
  children: React.ReactNode;
}

/**
 * 路由守卫：未登录时重定向到 /login，并附带当前路径作为 redirect 参数。
 */
export function AuthGuard({ children }: AuthGuardProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to={`/login?redirect=${encodeURIComponent(location.pathname)}`} replace />;
  }

  return <>{children}</>;
}