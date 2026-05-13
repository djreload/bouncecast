import React, { ReactElement } from 'react';
import dynamic from 'next/dynamic';
import { Button, Card, Col, Empty, Row, Space, Statistic, Table, Tag, Typography } from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';

const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const MailOutlined = dynamic(() => import('@ant-design/icons/MailOutlined'), { ssr: false });
const CustomerServiceOutlined = dynamic(() => import('@ant-design/icons/CustomerServiceOutlined'), {
  ssr: false,
});

const { Title, Text } = Typography;

const columns = [
  {
    title: 'DJ',
    dataIndex: 'name',
    key: 'name',
    render: (name, record) => (
      <Space direction="vertical" size={0}>
        <Text strong>{name}</Text>
        <Text type="secondary">{record.handle}</Text>
      </Space>
    ),
  },
  {
    title: 'Role',
    dataIndex: 'role',
    key: 'role',
    render: role => <Tag color={role === 'Owner' ? 'gold' : 'blue'}>{role}</Tag>,
  },
  {
    title: 'Stream Key',
    dataIndex: 'streamKey',
    key: 'streamKey',
    render: streamKey => <Tag color={streamKey === 'Ready' ? 'green' : 'default'}>{streamKey}</Tag>,
  },
  {
    title: 'Notifications',
    dataIndex: 'notifications',
    key: 'notifications',
  },
];

const streamerRows = [
  {
    key: 'foundation',
    name: 'Primary channel owner',
    handle: 'Current single-admin BounceCast account',
    role: 'Owner',
    streamKey: 'Ready',
    notifications: 'Browser, webhooks, federation',
  },
];

export default function Streamers() {
  return (
    <div className="bouncecast-admin-page">
      <div className="studio-hero">
        <div>
          <Text className="studio-eyebrow">Studio roster</Text>
          <Title level={1}>DJ streamer management</Title>
          <Text>
            Prepare BounceCast for multiple DJ logins, per-streamer stream keys, role-based admin
            access, and go-live notification ownership.
          </Text>
        </div>
        <Button type="primary" icon={<UserAddOutlined />} disabled>
          Add streamer
        </Button>
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Active streamer accounts"
              value={1}
              prefix={<CustomerServiceOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic title="Per-DJ stream keys" value={0} prefix={<KeyOutlined />} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic title="Email routes" value={0} prefix={<MailOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card title="Streamer accounts" className="studio-panel">
        <Table columns={columns} dataSource={streamerRows} pagination={false} />
      </Card>

      <Card className="studio-panel">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="Multi-login backend tables, session auth, and streamer-scoped stream keys are the next implementation step."
        />
      </Card>
    </div>
  );
}

Streamers.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
