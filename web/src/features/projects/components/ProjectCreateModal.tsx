import { Modal, Form, Input, Select } from 'antd';
import { useCreateProject } from '../hooks';

interface ProjectCreateModalProps {
  open: boolean;
  onClose: () => void;
}

interface FormValues {
  name: string;
  description?: string;
  environment: 'development' | 'staging' | 'production';
}

/**
 * 创建项目弹窗。
 */
export function ProjectCreateModal({ open, onClose }: ProjectCreateModalProps) {
  const createProject = useCreateProject();
  const [form] = Form.useForm<FormValues>();

  const handleSubmit = async (values: FormValues) => {
    await createProject.mutateAsync(values);
    form.resetFields();
    onClose();
  };

  const handleCancel = () => {
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title="创建项目"
      open={open}
      onCancel={handleCancel}
      onOk={() => form.submit()}
      confirmLoading={createProject.isPending}
      width={520}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{ environment: 'development' }}
      >
        <Form.Item
          name="name"
          label="项目名称"
          rules={[
            { required: true, message: '请输入项目名称' },
            { max: 255, message: '项目名称不超过 255 个字符' },
          ]}
        >
          <Input placeholder="我的项目" />
        </Form.Item>

        <Form.Item name="description" label="项目描述">
          <Input.TextArea placeholder="项目描述（可选）" rows={3} />
        </Form.Item>

        <Form.Item name="environment" label="环境">
          <Select
            options={[
              { value: 'development', label: '开发环境' },
              { value: 'staging', label: '预发布环境' },
              { value: 'production', label: '生产环境' },
            ]}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
