import { Modal, Form, Input } from 'antd';
import { useCreateProject } from '../hooks';

interface ProjectCreateModalProps {
  open: boolean;
  onClose: () => void;
}

interface FormValues {
  project_id: string;
  name: string;
  description?: string;
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
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          name="project_id"
          label="项目标识"
          rules={[
            { required: true, message: '请输入项目标识' },
            { pattern: /^[a-zA-Z0-9_-]+$/, message: '仅允许字母、数字、下划线和连字符' },
          ]}
        >
          <Input placeholder="my-project" />
        </Form.Item>

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
      </Form>
    </Modal>
  );
}