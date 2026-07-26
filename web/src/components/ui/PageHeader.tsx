import { Breadcrumb, Typography, Space } from 'antd';
import { ReactNode } from 'react';
import { Link } from 'react-router-dom';

const { Title } = Typography;

interface BreadcrumbItem {
  title: string;
  path?: string;
}

interface PageHeaderProps {
  title: ReactNode;
  breadcrumbs?: BreadcrumbItem[];
  extra?: ReactNode;
}

export function PageHeader({ title, breadcrumbs, extra }: PageHeaderProps) {
  return (
    <div style={{ marginBottom: 24 }}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumb
          style={{ marginBottom: 8 }}
          items={breadcrumbs.map((item) => ({
            title: item.path ? <Link to={item.path}>{item.title}</Link> : item.title,
          }))}
        />
      )}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>{title}</Title>
        {extra && <Space>{extra}</Space>}
      </div>
    </div>
  );
}