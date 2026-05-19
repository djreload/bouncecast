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
  BOUNCECAST_ADMIN_SESSION,
  BOUNCECAST_STREAMERS,
  BOUNCECAST_STREAMER_PASSWORD,
  BOUNCECAST_STREAMER_UPDATE,
  BOUNCECAST_STREAM_KEYS,
  BOUNCECAST_STREAM_KEY_REVOKE,
  fetchData,
} from '../../utils/apis';

const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const LockOutlined = dynamic(() => import('@ant-design/icons/LockOutlined'), { ssr: false });
const EditOutlined = dynamic(() => import('@ant-design/icons/EditOutlined'), { ssr: false });
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });
const CustomerServiceOutlined = dynamic(() => import('@ant-design/icons/CustomerServiceOutlined'), {
  ssr: false,
});
const StopOutlined = dynamic(() => import('@ant-design/icons/StopOutlined'), { ssr: false });
const CheckCircleOutlined = dynamic(() => import('@ant-design/icons/CheckCircleOutlined'), {
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
  avatarUrl?: string;
  bio?: string;
  genres?: string[];
  socialLinks?: { label: string; url: string }[];
  heroImageUrl?: string;
  passwordSet: boolean;
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

type AdminSession = {
  role: string;
  owner: boolean;
  admin: boolean;
};

const statusColor = {
  inactive: 'gold',
  active: 'green',
  invited: 'gold',
  disabled: 'default',
};

const columns = (
  openPasswordModal: (streamer: Streamer) => void,
  openProfileModal: (streamer: Streamer) => void,
  updateStreamerStatus: (streamer: Streamer, status: string) => void,
) => [
  {
    title: 'DJ',
    dataIndex: 'displayName',
    key: 'name',
    render: (name, record) => (
      <Space direction="vertical" size={0}>
        <Text strong>{name}</Text>
        <Text type="secondary">@{record.handle}</Text>
        {record.genres?.length > 0 && (
          <Text type="secondary">{record.genres.slice(0, 3).join(', ')}</Text>
        )}
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
    render: status => <Tag color={statusColor[status] || 'default'}>{status}</Tag>,
  },
  {
    title: 'Dashboard',
    dataIndex: 'passwordSet',
    key: 'passwordSet',
    render: passwordSet => (
      <Tag color={passwordSet ? 'green' : 'gold'}>
        {passwordSet ? 'password set' : 'needs password'}
      </Tag>
    ),
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
  {
    title: 'Actions',
    key: 'actions',
    render: (_, streamer) => (
      <Space>
        <Button size="small" icon={<EditOutlined />} onClick={() => openProfileModal(streamer)}>
          Profile
        </Button>
        <Button size="small" icon={<LockOutlined />} onClick={() => openPasswordModal(streamer)}>
          Password
        </Button>
        {streamer.status === 'active' ? (
          <Button
            size="small"
            danger
            icon={<StopOutlined />}
            onClick={() => updateStreamerStatus(streamer, 'disabled')}
          >
            Disable
          </Button>
        ) : (
          <Button
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => updateStreamerStatus(streamer, 'active')}
          >
            Activate
          </Button>
        )}
      </Space>
    ),
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

function normalizeStreamerFormPayload(values) {
  return {
    ...values,
    genres: String(values.genres || '')
      .split(',')
      .map(genre => genre.trim())
      .filter(Boolean),
    socialLinks: String(values.socialLinks || '')
      .split('\n')
      .map(line => {
        const [label, ...urlParts] = line.split('|');
        return { label: label?.trim(), url: urlParts.join('|').trim() };
      })
      .filter(link => link.label || link.url),
  };
}

type StreamerKeysTableProps = {
  streamer: Streamer;
  streamKeys: StreamKey[];
  openKeyModal: (streamer: Streamer) => void;
  revokeStreamKey: (id: number) => void;
  canManage: boolean;
};

const StreamerKeysTable = ({
  streamer,
  streamKeys,
  openKeyModal,
  revokeStreamKey,
  canManage,
}: StreamerKeysTableProps) => {
  const keys = streamKeys.filter(key => key.streamerId === streamer.id);
  return (
    <div className="studio-nested-table">
      {canManage && (
        <Space className="studio-table-actions">
          <Button size="small" icon={<KeyOutlined />} onClick={() => openKeyModal(streamer)}>
            Create stream key
          </Button>
        </Space>
      )}
      <Table
        columns={
          canManage
            ? streamKeyColumns(revokeStreamKey)
            : streamKeyColumns(revokeStreamKey).filter(column => column.key !== 'action')
        }
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
  const [adminSession, setAdminSession] = useState<AdminSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [selectedStreamer, setSelectedStreamer] = useState<Streamer | null>(null);
  const [newStreamKey, setNewStreamKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [keyForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [profileForm] = Form.useForm();
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [profileModalOpen, setProfileModalOpen] = useState(false);
  const [passwordStreamer, setPasswordStreamer] = useState<Streamer | null>(null);
  const [profileStreamer, setProfileStreamer] = useState<Streamer | null>(null);

  const loadStreamers = async () => {
    setLoading(true);
    try {
      const [streamerResult, keyResult, adminSessionResult] = await Promise.all([
        fetchData(BOUNCECAST_STREAMERS),
        fetchData(BOUNCECAST_STREAM_KEYS),
        fetchData(BOUNCECAST_ADMIN_SESSION),
      ]);
      setStreamers(streamerResult || []);
      setStreamKeys(keyResult || []);
      setAdminSession(adminSessionResult || null);
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

  const canManageStreamerAccess = Boolean(adminSession?.admin);

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

  const openPasswordModal = (streamer: Streamer) => {
    setPasswordStreamer(streamer);
    passwordForm.resetFields();
    setPasswordModalOpen(true);
  };

  const setStreamerPassword = async () => {
    if (!passwordStreamer) {
      return;
    }
    const values = await passwordForm.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_STREAMER_PASSWORD, {
        method: 'POST',
        data: {
          id: passwordStreamer.id,
          password: values.password,
        },
      });
      setPasswordModalOpen(false);
      await loadStreamers();
    } finally {
      setSaving(false);
    }
  };

  const updateStreamerStatus = async (streamer: Streamer, status: string) => {
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_STREAMER_UPDATE, {
        method: 'POST',
        data: {
          id: streamer.id,
          displayName: streamer.displayName,
          handle: streamer.handle,
          email: streamer.email,
          role: streamer.role,
          status,
          avatarUrl: streamer.avatarUrl,
          bio: streamer.bio,
          genres: streamer.genres || [],
          socialLinks: streamer.socialLinks || [],
          heroImageUrl: streamer.heroImageUrl,
        },
      });
      await loadStreamers();
    } finally {
      setSaving(false);
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
        data: normalizeStreamerFormPayload(values),
      });
      form.resetFields();
      setModalOpen(false);
      await loadStreamers();
    } finally {
      setSaving(false);
    }
  };

  const openProfileModal = (streamer: Streamer) => {
    setProfileStreamer(streamer);
    profileForm.setFieldsValue({
      ...streamer,
      genres: (streamer.genres || []).join(', '),
      socialLinks: (streamer.socialLinks || []).map(link => `${link.label}|${link.url}`).join('\n'),
    });
    setProfileModalOpen(true);
  };

  const saveProfile = async () => {
    if (!profileStreamer) {
      return;
    }
    const values = await profileForm.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_STREAMER_UPDATE, {
        method: 'POST',
        data: {
          id: profileStreamer.id,
          role: profileStreamer.role,
          status: profileStreamer.status,
          email: profileStreamer.email,
          handle: profileStreamer.handle,
          ...normalizeStreamerFormPayload(values),
        },
      });
      setProfileModalOpen(false);
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
        {canManageStreamerAccess && (
          <Space>
            <Button icon={<KeyOutlined />} onClick={() => openKeyModal()}>
              New key
            </Button>
            <Button type="primary" icon={<UserAddOutlined />} onClick={() => setModalOpen(true)}>
              Add streamer
            </Button>
          </Space>
        )}
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Active streamer accounts"
              value={streamers.filter(streamer => streamer.status === 'active').length}
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
            <Statistic
              title="Dashboard logins"
              value={streamers.filter(streamer => streamer.passwordSet).length}
              prefix={<LockOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card title="Streamer accounts" className="studio-panel">
        <Table
          columns={
            canManageStreamerAccess
              ? columns(openPasswordModal, openProfileModal, updateStreamerStatus)
              : columns(openPasswordModal, openProfileModal, updateStreamerStatus).filter(
                  column => column.key !== 'actions',
                )
          }
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
                openKeyModal={canManageStreamerAccess ? openKeyModal : () => undefined}
                revokeStreamKey={canManageStreamerAccess ? revokeStreamKey : () => undefined}
                canManage={canManageStreamerAccess}
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
        <Form
          form={form}
          layout="vertical"
          initialValues={{ role: 'streamer', status: 'inactive' }}
        >
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
          <Form.Item name="status" label="Status">
            <Select
              options={[
                { label: 'Inactive', value: 'inactive' },
                { label: 'Active', value: 'active' },
                { label: 'Invited', value: 'invited' },
                { label: 'Disabled', value: 'disabled' },
              ]}
            />
          </Form.Item>
          <Form.Item name="password" label="Dashboard password">
            <Input.Password placeholder="Optional for now" />
          </Form.Item>
          <Form.Item name="avatarUrl" label="Avatar URL">
            <Input placeholder="/public/profiles/dj.png" />
          </Form.Item>
          <Form.Item name="heroImageUrl" label="Hero image URL">
            <Input placeholder="Optional wide profile image" />
          </Form.Item>
          <Form.Item name="bio" label="Bio">
            <Input.TextArea rows={3} maxLength={600} />
          </Form.Item>
          <Form.Item name="genres" label="Genres">
            <Input placeholder="House, Garage, DnB" />
          </Form.Item>
          <Form.Item name="socialLinks" label="Social links">
            <Input.TextArea
              rows={3}
              placeholder={
                'Instagram|https://instagram.com/name\nMixcloud|https://mixcloud.com/name'
              }
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={profileStreamer ? `Edit profile for ${profileStreamer.displayName}` : 'Edit profile'}
        open={profileModalOpen}
        onCancel={() => setProfileModalOpen(false)}
        onOk={saveProfile}
        confirmLoading={saving}
        okText="Save profile"
      >
        <Form form={profileForm} layout="vertical">
          <Form.Item
            name="displayName"
            label="Display name"
            rules={[{ required: true, message: 'Add a display name' }]}
          >
            <Input maxLength={80} />
          </Form.Item>
          <Form.Item
            name="handle"
            label="Handle"
            rules={[{ required: true, message: 'Add a handle' }]}
          >
            <Input maxLength={32} prefix="@" />
          </Form.Item>
          <Form.Item name="email" label="Email">
            <Input />
          </Form.Item>
          <Form.Item name="avatarUrl" label="Avatar URL">
            <Input maxLength={500} />
          </Form.Item>
          <Form.Item name="heroImageUrl" label="Hero image URL">
            <Input maxLength={500} />
          </Form.Item>
          <Form.Item name="bio" label="Bio">
            <Input.TextArea rows={4} maxLength={600} />
          </Form.Item>
          <Form.Item name="genres" label="Genres">
            <Input placeholder="House, Garage, DnB" />
          </Form.Item>
          <Form.Item name="socialLinks" label="Social links">
            <Input.TextArea
              rows={4}
              placeholder={
                'Instagram|https://instagram.com/name\nMixcloud|https://mixcloud.com/name'
              }
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          passwordStreamer ? `Set password for ${passwordStreamer.displayName}` : 'Set password'
        }
        open={passwordModalOpen}
        onCancel={() => setPasswordModalOpen(false)}
        onOk={setStreamerPassword}
        confirmLoading={saving}
        okText="Save password"
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="password"
            label="New dashboard password"
            rules={[
              { required: true, message: 'Add a dashboard password' },
              { min: 8, message: 'Use at least 8 characters' },
            ]}
          >
            <Input.Password autoComplete="new-password" placeholder="At least 8 characters" />
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
