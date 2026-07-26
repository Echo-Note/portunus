import { Table, TableProps } from 'antd';
import type { TablePaginationConfig } from 'antd';
import { ReactNode } from 'react';
import { EmptyState } from './EmptyState';
import { ErrorResult } from './ErrorResult';
import type { ColumnsType } from 'antd/es/table';

/**
 * DataTable — 通用 CRUD 表格组件
 *
 * 封装了表格的三种标准状态：
 * - loading：加载中（Table 原生 loading）
 * - error：错误时展示 ErrorResult + 重试按钮
 * - empty：空数据时展示 EmptyState
 *
 * 同时支持可选的 toolbar 插槽，用于放置"创建"等操作按钮。
 */
interface DataTableProps<T extends object> extends Omit<TableProps<T>, 'loading' | 'columns'> {
  /** 列定义 */
  columns: ColumnsType<T>;
  /** 是否加载中 */
  loading?: boolean;
  /** 是否出错 */
  error?: boolean;
  /** 错误重试回调 */
  onRetry?: () => void;
  /** 空数据提示文案 */
  emptyText?: string;
  /** 表格上方的工具栏（如"创建"按钮） */
  toolbar?: ReactNode;
  /** 分页配置（默认 20 条/页，支持切换每页条数） */
  pagination?: TablePaginationConfig | false;
  /** 表格尺寸 */
  size?: 'small' | 'middle' | 'large';
}

export function DataTable<T extends object>({
  columns,
  loading = false,
  error = false,
  onRetry,
  emptyText = '暂无数据',
  toolbar,
  pagination,
  size,
  ...rest
}: DataTableProps<T>) {
  // 错误状态
  if (error) {
    return <ErrorResult onRetry={onRetry} />;
  }

  // 默认分页配置
  const defaultPagination: TablePaginationConfig = {
    defaultPageSize: 20,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
  };

  return (
    <div>
      {toolbar && (
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          {toolbar}
        </div>
      )}
      <Table<T>
        columns={columns}
        loading={loading}
        size={size}
        locale={{ emptyText: <EmptyState description={emptyText} /> }}
        pagination={pagination === false ? false : { ...defaultPagination, ...pagination }}
        {...rest}
      />
    </div>
  );
}