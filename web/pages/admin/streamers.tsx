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
import {
  BOUNCECAST_STREAMERS,
  BOUNCECAST_STREAM_KEYS,
  BOUNCECAST_STREAM_KEY_REVOKE,
  fetchData,
} from '../../utils/apis';

const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const MailOutlined = dynamic(() => import('@ant-design/icons/MailOutlined'), { ssr: false });
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });
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
  streamKeyCount: number;
};

type StreamKey = {
  id: number;
  streamerId: number;
  label: string;
  enabled: boolean;
  createdAt: string;
  lastUsedAt?: string;
  revokedAt?: string;
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
    title: 'Stream Keys',
    dataIndex: 'streamKeyCount',
    key: 'streamKeyCount',
    render: streamKeyCount => (
      <Tag color={streamKeyCount > 0 ? 'green' : 'default'}>{streamKeyCount}</Tag>
    ),
  },
  {
    title: 'Email',
    dataIndex: 'email',
    key: 'email',
    render: email => email || 'Not set',
  },
];

const streamKeyColumns = (revokeStreamKey: (id: number) => void) => [
  {
    title: 'Label',
    dataIndex: 'label',
    key: 'label',
    render: label => label || 'OBS key',
  },
  {
    title: 'Created',
    dataIndex: 'createdAt',
    key: 'createdAt',
    render: createdAt => new Date(createdAt).toLocaleString(),
  },
  {
    title: 'Last used',
    dataIndex: 'lastUsedAt',
    key: 'lastUsedAt',
    render: lastUsedAt => (lastUsedAt ? new Date(lastUsedAt).toLocaleString() : 'Never'),
  },
  {
    title: 'Status',
    key: 'status',
    render: (_, key) => (
      <Tag color={key.enabled && !key.revokedAt ? 'green' : 'default'}>
        {key.enabled && !key.revokedAt ? 'active' : 'revoked'}
      </Tag>
    ),
  },
  {
    title: 'Action',
    key: 'action',
    render: (_, key) =>
      key.enabled && !key.revokedAt ? (
        <Button size="small" danger onClick={() => revokeStreamKey(key.id)}>
          Revoke
        </Button>
      ) : null,
  },
];

type StreamerKeysTableProps = {
  streamer: Streamer;
  streamKeys: StreamKey[];
  openKeyModal: (streamer: Streamer) => void;
  revokeStreamKey: (id: number) => void;
};

const StreamerKeysTable = ({
  streamer,
  streamKeys,
  openKeyModal,
  revokeStreamKey,
}: StreamerKeysTableProps) => {
  const keys = streamKeys.filter(key => key.streamerId === streamer.id);
  return (
    <div className="studio-nested-table">
      <Space className="studio-table-actions">
        <Button size="small" icon={<KeyOutlined />} onClick={() => openKeyModal(streamer)}>
          Create stream key
        </Button>
      </Space>
      <Table
        columns={streamKeyColumns(revokeStreamKey)}
        dataSource={keys}
        rowKey="id"
        pagination={false}
        size="small"
      />
    </div>
  );
};

export default function Streamers() {
  const [streamers, setStreamers] = useState<Streamer[]>([]);
  const [streamKeys, setStreamKeys] = useState<StreamKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [selectedStreamer, setSelectedStreamer] = useState<Streamer | null>(null);
  const [newStreamKey, setNewStreamKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [keyForm] = Form.useForm();

  const loadStreamers = async () => {
    setLoading(true);
    try {
      const [streamerResult, keyResult] = await Promise.all([
        fetchData(BOUNCECAST_STREAMERS),
        fetchData(BOUNCECAST_STREAM_KEYS),
      ]);
      setStreamers(streamerResult || []);
      setStreamKeys(keyResult || []);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const createStreamKey = async () => {
    const values = await keyForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchData(BOUNCECAST_STREAM_KEYS, {
        method: 'POST',
        data: {
          streamerId: values.streamerId,
          label: values.label,
        },
      });
      setNewStreamKey(result.streamKey);
      await loadStreamers();
    } finally {
      setSaving(false);
    }
  };

  const revokeStreamKey = async (id: number) => {
    await fetchData(BOUNCECAST_STREAM_KEY_REVOKE, {
      method: 'POST',
      data: { id },
    });
    await loadStreamers();
  };

  const openKeyModal = (streamer?: Streamer) => {
    setSelectedStreamer(streamer || null);
    setNewStreamKey('');
    keyForm.resetFields();
    keyForm.setFieldsValue({
      streamerId: streamer?.id,
      label: streamer ? `${streamer.displayName} OBS key` : '',
    });
    setKeyModalOpen(true);
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
        <Space>
          <Button icon={<KeyOutlined />} onClick={() => openKeyModal()}>
            New key
          </Button>
          <Button type="primary" icon={<UserAddOutlined />} onClick={() => setModalOpen(true)}>
            Add streamer
          </Button>
        </Space>
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
            <Statistic
              title="Per-DJ stream keys"
              value={streamKeys.length}
              prefix={<KeyOutlined />}
            />
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
          expandable={{
            // Ant Design requires a render callback here so the expanded row can receive the table record.
            // eslint-disable-next-line react/no-unstable-nested-components
            expandedRowRender: streamer => (
              <StreamerKeysTable
                streamer={streamer}
                streamKeys={streamKeys}
                openKeyModal={openKeyModal}
                revokeStreamKey={revokeStreamKey}
              />
            ),
          }}
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

      <Modal
        title={
          selectedStreamer ? `New stream key for ${selectedStreamer.displayName}` : 'New stream key'
        }
        open={keyModalOpen}
        onCancel={() => setKeyModalOpen(false)}
        onOk={newStreamKey ? () => setKeyModalOpen(false) : createStreamKey}
        confirmLoading={saving}
        okText={newStreamKey ? 'Done' : 'Create key'}
      >
        {newStreamKey ? (
          <Space direction="vertical" className="studio-secret-box">
            <Text strong>Copy this RTMP stream key now. It will not be shown again.</Text>
            <Input.TextArea value={newStreamKey} rows={3} readOnly />
            <Button
              icon={<CopyOutlined />}
              onClick={() => navigator.clipboard?.writeText(newStreamKey)}
            >
              Copy key
            </Button>
          </Space>
        ) : (
          <Form form={keyForm} layout="vertical">
            <Form.Item
              name="streamerId"
              label="Streamer"
              rules={[{ required: true, message: 'Choose a streamer' }]}
            >
              <Select
                placeholder="Assign to DJ"
                options={streamers.map(streamer => ({
                  label: streamer.displayName,
                  value: streamer.id,
                }))}
              />
            </Form.Item>
            <Form.Item name="label" label="Label">
              <Input placeholder="OBS laptop key" />
            </Form.Item>
          </Form>
        )}
      </Modal>
    </div>
  );
}

Streamers.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
