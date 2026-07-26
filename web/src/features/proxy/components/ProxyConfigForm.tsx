import { Form, Select, Input, Button, Card, Spin, message } from 'antd';
import { useEffect } from 'react';
import { useProxyConfig, useUpdateProxy } from '../hooks';
import { LBPolicy, UpdateProxyInput } from '../types';
import { ErrorResult } from '../../../components/ui/ErrorResult';

interface ProxyConfigFormProps {
  projectId: string;
  domainId: string;
}

const LB_POLICY_OPTIONS: { value: LBPolicy; label: string }[] = [
  { value: 'random', label: '随机 (random)' },
  { value: 'round_robin', label: '轮询 (round_robin)' },
  { value: 'least_conn', label: '最少连接 (least_conn)' },
  { value: 'ip_hash', label: 'IP 哈希 (ip_hash)' },
  { value: 'uri_hash', label: 'URI 哈希 (uri_hash)' },
];

/**
 * 代理配置表单：LB 策略、健康检查、超时配置。
 */
export function ProxyConfigForm({ projectId, domainId }: ProxyConfigFormProps) {
  const { data: proxyConfig, isLoading, isError, refetch } = useProxyConfig(projectId, domainId);
  const updateProxy = useUpdateProxy(projectId, domainId);
  const [form] = Form.useForm<UpdateProxyInput>();

  useEffect(() => {
    if (proxyConfig) {
      form.setFieldsValue({
        lb_policy: proxyConfig.lb_policy,
        health_check_uri: proxyConfig.health_check_uri,
        health_check_interval: proxyConfig.health_check_interval,
        timeout: proxyConfig.timeout,
      });
    }
  }, [proxyConfig, form]);

  const handleSubmit = async (values: UpdateProxyInput) => {
    await updateProxy.mutateAsync(values);
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 24 }}>
        <Spin />
      </div>
    );
  }

  if (isError) {
    return <ErrorResult onRetry={() => refetch()} />;
  }

  return (
    <Card title="代理配置" size="small">
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item name="lb_policy" label="负载均衡策略">
          <Select options={LB_POLICY_OPTIONS} />
        </Form.Item>

        <Form.Item name="health_check_uri" label="健康检查路径">
          <Input placeholder="/health" />
        </Form.Item>

        <Form.Item name="health_check_interval" label="健康检查间隔">
          <Input placeholder="30s" />
        </Form.Item>

        <Form.Item name="timeout" label="超时时间">
          <Input placeholder="30s" />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={updateProxy.isPending}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
}
