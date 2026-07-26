import { Result, Button } from 'antd';

interface ErrorResultProps {
  message?: string;
  onRetry?: () => void;
}

export function ErrorResult({ message = '加载失败，请重试', onRetry }: ErrorResultProps) {
  return (
    <Result
      status="error"
      title="加载失败"
      subTitle={message}
      extra={onRetry && <Button type="primary" onClick={onRetry}>重试</Button>}
    />
  );
}