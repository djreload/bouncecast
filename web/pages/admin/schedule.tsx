import React, { ReactElement } from 'react';
import dynamic from 'next/dynamic';
import {
  Button,
  Card,
  Col,
  Empty,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';

const CalendarOutlined = dynamic(() => import('@ant-design/icons/CalendarOutlined'), {
  ssr: false,
});
const BellOutlined = dynamic(() => import('@ant-design/icons/BellOutlined'), { ssr: false });
const ClockCircleOutlined = dynamic(() => import('@ant-design/icons/ClockCircleOutlined'), {
  ssr: false,
});
const ThunderboltOutlined = dynamic(() => import('@ant-design/icons/ThunderboltOutlined'), {
  ssr: false,
});

const { Title, Text } = Typography;

const columns = [
  {
    title: 'Set',
    dataIndex: 'title',
    key: 'title',
    render: (title, record) => (
      <Space direction="vertical" size={0}>
        <Text strong>{title}</Text>
        <Text type="secondary">{record.dj}</Text>
      </Space>
    ),
  },
  {
    title: 'Start',
    dataIndex: 'start',
    key: 'start',
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    render: status => <Tag color={status === 'Planned' ? 'purple' : 'default'}>{status}</Tag>,
  },
  {
    title: 'Alerts',
    dataIndex: 'alerts',
    key: 'alerts',
  },
];

const scheduleRows = [
  {
    key: 'planning',
    title: 'Weekly live DJ slot',
    dj: 'Primary channel owner',
    start: 'Schedule backend pending',
    status: 'Planned',
    alerts: 'Push, email, webhook',
  },
];

export default function Schedule() {
  return (
    <div className="bouncecast-admin-page">
      <div className="studio-hero">
        <div>
          <Text className="studio-eyebrow">Live calendar</Text>
          <Title level={1}>DJ schedule and go-live alerts</Title>
          <Text>
            Build toward scheduled sets, automatic live announcements, and push/email campaigns when
            a DJ connects to their assigned stream key.
          </Text>
        </div>
        <Button type="primary" icon={<CalendarOutlined />} disabled>
          New set
        </Button>
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card>
            <Statistic title="Upcoming sets" value={1} prefix={<ClockCircleOutlined />} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic title="Go-live automations" value={0} prefix={<ThunderboltOutlined />} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic title="Subscriber channels" value={0} prefix={<BellOutlined />} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={15}>
          <Card title="Schedule" className="studio-panel">
            <Table columns={columns} dataSource={scheduleRows} pagination={false} />
          </Card>
        </Col>
        <Col xs={24} lg={9}>
          <Card title="Automation flow" className="studio-panel">
            <Timeline>
              <Timeline.Item color="blue">DJ account owns a scheduled set</Timeline.Item>
              <Timeline.Item color="purple">
                Assigned stream key connects through RTMP
              </Timeline.Item>
              <Timeline.Item color="green">
                Push, email, webhook, and federation alerts are queued
              </Timeline.Item>
            </Timeline>
          </Card>
        </Col>
      </Row>

      <Card className="studio-panel">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="Persistent schedule tables, notification providers, and per-streamer live events are required before this can send real alerts."
        />
      </Card>
    </div>
  );
}

Schedule.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
