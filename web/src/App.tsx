import { Providers } from './app/providers';
import { Router } from './app/router';

/**
 * 应用根组件：组合全局 Provider 和路由。
 */
export default function App() {
  return (
    <Providers>
      <Router />
    </Providers>
  );
}