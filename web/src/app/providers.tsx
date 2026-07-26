import { ConfigProvider, message } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '../lib/query-client';

// 全局 message 配置
message.config({ top: 60, duration: 3 });

interface ProvidersProps {
  children: React.ReactNode;
}

/**
 * 全局 Provider 组合：Ant Design 主题 + 路由 + TanStack Query。
 */
export function Providers({ children }: ProvidersProps) {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#1677ff',
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          {children}
        </BrowserRouter>
      </QueryClientProvider>
    </ConfigProvider>
  );
}