import React, { ReactElement, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';
import { BOUNCECAST_STREAMERS, fetchData } from '../../utils/apis';

const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const MailOutlined = dynamic(() => import('@ant-design/icons/MailOutlined'), { ssr: false });
const CustomerServiceOutlined = dynamic(() => import('@ant-design/icons/CustomerServiceOutlined'), {
  ssr: false,
});

const { Title, Text } = Typography;

type Streamer = {
  id: number;
  displayName: string;
  handle: string;
  email: string;
  role: string;
  status: string;
};

const columns = [
  {
    title: 'DJ',
    dataIndex: 'displayName',
    key: 'name',
    render: (name, record) => (
      <Space direction="vertical" size={0}>
        <Text strong>{name}</Text>
        <Text type="secondary">@{record.handle}</Text>
      </Space>
    ),
  },
  {
    title: 'Role',
    dataIndex: 'role',
    key: 'role',
    render: role => <Tag color={role === 'owner' ? 'gold' : 'blue'}>{role}</Tag>,
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    render: status => <Tag color={status === 'active' ? 'green' : 'default'}>{status}</Tag>,
  },
  {
    title: 'Email',
    dataIndex: 'email',
    key: 'email',
    render: email => email || 'Not set',
  },
];

export default function Streamers() {
  const [streamers, setStreamers] = useState<Streamer[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const loadStreamers = async () => {
    setLoading(true);
    try {
      const result = await fetchData(BOUNCECAST_STREAMERS);
      setStreamers(result || []);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStreamers();
  }, []);

  const createStreamer = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_STREAMERS, {
        method: 'POST',
        data: values,
      });
      form.resetFields();
      setModalOpen(false);
      await loadStreamers();
    } finally {
      setSaving(false);
    }
  };

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
        <Button type="primary" icon={<UserAddOutlined />} onClick={() => setModalOpen(true)}>
          Add streamer
        </Button>
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Active streamer accounts"
              value={streamers.length}
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
        <Table
          columns={columns}
          dataSource={streamers}
          loading={loading}
          rowKey="id"
          pagination={false}
        />
      </Card>

      {!loading && streamers.length === 0 && (
        <Card className="studio-panel">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="Create the first DJ account shell to begin wiring per-streamer access."
          />
        </Card>
      )}

      <Modal
        title="Add streamer"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={createStreamer}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical" initialValues={{ role: 'streamer' }}>
          <Form.Item
            name="displayName"
            label="Display name"
            rules={[{ required: true, message: 'Add a display name' }]}
          >
            <Input placeholder="DJ name" />
          </Form.Item>
          <Form.Item
            name="handle"
            label="Handle"
            rules={[{ required: true, message: 'Add a handle' }]}
          >
            <Input placeholder="dj-name" prefix="@" />
          </Form.Item>
          <Form.Item name="email" label="Email">
            <Input placeholder="dj@example.com" />
          </Form.Item>
          <Form.Item name="role" label="Role">
            <Select
              options={[
                { label: 'Streamer', value: 'streamer' },
                { label: 'Manager', value: 'manager' },
                { label: 'Owner', value: 'owner' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

Streamers.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
