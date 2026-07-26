import { Modal, Form, Input, InputNumber } from 'antd';
import { useAddUpstream } from '../hooks';

interface UpstreamAddModalProps {
  open: boolean;
  projectId: string;
  domainId: string;
  onClose: () => void;
}

interface FormValues {
  dial_address: string;
  weight: number;
}

/**
 * 添加上游弹窗。
 */
export function UpstreamAddModal({ open, projectId, domainId, onClose }: UpstreamAddModalProps) {
  const addUpstream = useAddUpstream(projectId, domainId);
  const [form] = Form.useForm<FormValues>();

  const handleSubmit = async (values: FormValues) => {
    await addUpstream.mutateAsync(values);
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title="添加上游"
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={addUpstream.isPending}
      width={520}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{ weight: 1 }}
      >
        <Form.Item
          name="dial_address"
          label="上游地址"
          rules={[
            { required: true, message: '请输入上游地址' },
            { pattern: /^[a-zA-Z0-9.-]+:\d+$/, message: '请输入有效的地址格式（host:port）' },
          ]}
        >
          <Input placeholder="backend.example.com:8080" />
        </Form.Item>

        <Form.Item name="weight" label="权重">
          <InputNumber min={1} max={100} style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
