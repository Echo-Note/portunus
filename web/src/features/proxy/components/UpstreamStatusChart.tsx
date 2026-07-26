import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Card, Spin, Typography } from 'antd';
import { useUpstreamStatus } from '../hooks';

const { Text } = Typography;

interface UpstreamStatusChartProps {
  projectId: string;
  domainId: string;
}

const COLORS = {
  healthy: '#52c41a',
  unhealthy: '#ff4d4f',
  disabled: '#faad14',
  unknown: '#d9d9d9',
};

/**
 * 上游健康状态饼图。
 */
export function UpstreamStatusChart({ projectId, domainId }: UpstreamStatusChartProps) {
  const { data: statuses = [], isLoading } = useUpstreamStatus(projectId, domainId);

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 24 }}>
        <Spin />
      </div>
    );
  }

  if (statuses.length === 0) {
    return <Text type="secondary">暂无健康状态数据</Text>;
  }

  // 统计各状态数量
  const counts: Record<string, number> = {};
  statuses.forEach((s) => {
    const key = s.healthy ? 'healthy' : 'unhealthy';
    counts[key] = (counts[key] || 0) + 1;
  });

  const chartData = Object.entries(counts).map(([name, value]) => ({
    name: name === 'healthy' ? '健康' : '异常',
    value,
  }));

  return (
    <Card title="健康状态概览" size="small">
      <ResponsiveContainer width="100%" height={200}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            dataKey="value"
            label={({ name, value }) => `${name}: ${value}`}
          >
            {chartData.map((entry, index) => (
              <Cell
                key={`cell-${index}`}
                fill={COLORS[entry.name === '健康' ? 'healthy' : 'unhealthy']}
              />
            ))}
          </Pie>
          <Tooltip />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  );
}
