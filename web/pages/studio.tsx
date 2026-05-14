import React, { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  BOUNCECAST_STUDIO_LIVE_EVENTS,
  BOUNCECAST_STUDIO_LOGIN,
  BOUNCECAST_STUDIO_LOGOUT,
  BOUNCECAST_STUDIO_ME,
  BOUNCECAST_STUDIO_SCHEDULE,
  BOUNCECAST_STUDIO_STREAM_KEYS,
  BOUNCECAST_STUDIO_STREAM_KEY_REVOKE,
  fetchStudioData,
} from '../utils/apis';

const CalendarOutlined = dynamic(() => import('@ant-design/icons/CalendarOutlined'), {
  ssr: false,
});
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const LogoutOutlined = dynamic(() => import('@ant-design/icons/LogoutOutlined'), { ssr: false });
const PlayCircleOutlined = dynamic(() => import('@ant-design/icons/PlayCircleOutlined'), {
  ssr: false,
});
const ReloadOutlined = dynamic(() => import('@ant-design/icons/ReloadOutlined'), { ssr: false });

const { Text } = Typography;

const studioTokenStorageKey = 'bouncecastStudioToken';

type Streamer = {
  id: number;
  displayName: string;
  handle: string;
  email: string;
  role: string;
  status: string;
};

type StudioSession = {
  token?: string;
  expiresAt: string;
  streamer: Streamer;
};

type ScheduleItem = {
  id: number;
  title: string;
  description: string;
  startsAt: string;
  endsAt?: string;
  timezone: string;
  status: string;
  notifyEmail: boolean;
  notifyPush: boolean;
  notifyWebhook: boolean;
};

type StreamKey = {
  id: number;
  label: string;
  enabled: boolean;
  createdAt: string;
  lastUsedAt?: string;
  revokedAt?: string;
};

type LiveEvent = {
  id: number;
  scheduleTitle: string;
  startedAt: string;
  endedAt?: string;
  status: string;
  notificationState: string;
};

function formatDate(value?: string) {
  if (!value) {
    return 'Not set';
  }
  return new Date(value).toLocaleString();
}

function statusColor(status: string) {
  if (status === 'live' || status === 'active' || status === 'sent') {
    return 'green';
  }
  if (status === 'planned' || status === 'queued') {
    return 'gold';
  }
  if (status === 'failed') {
    return 'red';
  }
  return 'default';
}

export default function Studio() {
  const [token, setToken] = useState('');
  const [session, setSession] = useState<StudioSession | null>(null);
  const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
  const [streamKeys, setStreamKeys] = useState<StreamKey[]>([]);
  const [liveEvents, setLiveEvents] = useState<LiveEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [newStreamKey, setNewStreamKey] = useState('');
  const [loginForm] = Form.useForm();
  const [keyForm] = Form.useForm();

  const loadStudio = async (activeToken: string) => {
    setLoading(true);
    try {
      const [sessionResult, scheduleResult, keysResult, eventsResult] = await Promise.all([
        fetchStudioData(BOUNCECAST_STUDIO_ME, activeToken),
        fetchStudioData(BOUNCECAST_STUDIO_SCHEDULE, activeToken),
        fetchStudioData(BOUNCECAST_STUDIO_STREAM_KEYS, activeToken),
        fetchStudioData(BOUNCECAST_STUDIO_LIVE_EVENTS, activeToken),
      ]);
      setSession(sessionResult);
      setSchedule(scheduleResult || []);
      setStreamKeys(keysResult || []);
      setLiveEvents(eventsResult || []);
      setToken(activeToken);
    } catch (error) {
      localStorage.removeItem(studioTokenStorageKey);
      setToken('');
      setSession(null);
      setSchedule([]);
      setStreamKeys([]);
      setLiveEvents([]);
      if (activeToken) {
        message.error(error instanceof Error ? error.message : 'Studio session expired');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const savedToken = localStorage.getItem(studioTokenStorageKey) || '';
    if (savedToken) {
      loadStudio(savedToken);
      return;
    }
    setLoading(false);
  }, []);

  const login = async () => {
    const values = await loginForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchStudioData(BOUNCECAST_STUDIO_LOGIN, undefined, {
        method: 'POST',
        data: values,
      });
      localStorage.setItem(studioTokenStorageKey, result.token);
      await loadStudio(result.token);
      loginForm.resetFields();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to log in');
    } finally {
      setSaving(false);
    }
  };

  const logout = async () => {
    if (token) {
      try {
        await fetchStudioData(BOUNCECAST_STUDIO_LOGOUT, token, { method: 'POST' });
      } catch (error) {
        console.error(error);
      }
    }
    localStorage.removeItem(studioTokenStorageKey);
    setToken('');
    setSession(null);
    setSchedule([]);
    setStreamKeys([]);
    setLiveEvents([]);
  };

  const createStreamKey = async () => {
    const values = await keyForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchStudioData(BOUNCECAST_STUDIO_STREAM_KEYS, token, {
        method: 'POST',
        data: values,
      });
      setNewStreamKey(result.streamKey);
      keyForm.resetFields();
      await loadStudio(token);
    } finally {
      setSaving(false);
    }
  };

  const revokeStreamKey = async (id: number) => {
    setSaving(true);
    try {
      await fetchStudioData(BOUNCECAST_STUDIO_STREAM_KEY_REVOKE, token, {
        method: 'POST',
        data: { id },
      });
      await loadStudio(token);
    } finally {
      setSaving(false);
    }
  };

  const activeKeys = streamKeys.filter(key => key.enabled && !key.revokedAt);
  const nextSet = schedule.find(item => item.status === 'planned') || schedule[0];

  if (!session && !loading) {
    return (
      <main className="bouncecast-studio-dashboard">
        <div className="studio-shell studio-login-shell">
          <section className="studio-live-strip">
            <div className="studio-live-copy">
              <h1>BounceCast Studio</h1>
              <Text className="studio-brand-subtitle">
                DJ access for scheduled live sets, stream keys, and live history.
              </Text>
            </div>
          </section>
          <section className="studio-login-panel">
            <div className="studio-brand-lockup">
              <img src="/logo" alt="BounceCast" />
              <div>
                <p className="studio-brand-title">Studio login</p>
                <Text className="studio-brand-subtitle">BounceCast DJ dashboard</Text>
              </div>
            </div>
            <Card className="studio-login-card">
              <Form form={loginForm} layout="vertical">
                <Form.Item
                  name="login"
                  label="Handle or email"
                  rules={[{ required: true, message: 'Enter your handle or email' }]}
                >
                  <Input autoComplete="username" placeholder="@dj-name" />
                </Form.Item>
                <Form.Item
                  name="password"
                  label="Password"
                  rules={[{ required: true, message: 'Enter your password' }]}
                >
                  <Input.Password autoComplete="current-password" />
                </Form.Item>
                <Button type="primary" block loading={saving} onClick={login}>
                  Log in
                </Button>
              </Form>
            </Card>
          </section>
        </div>
      </main>
    );
  }

  return (
    <main className="bouncecast-studio-dashboard">
      <div className="studio-shell">
        <header className="studio-dashboard-header">
          <div>
            <p className="studio-dashboard-title">
              {session ? `${session.streamer.displayName} Studio` : 'BounceCast Studio'}
            </p>
            {session && (
              <div className="studio-session-meta">
                <Tag color="blue">@{session.streamer.handle}</Tag>
                <Tag color={statusColor(session.streamer.status)}>{session.streamer.status}</Tag>
                <Tag>{session.streamer.role}</Tag>
              </div>
            )}
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => loadStudio(token)}>
              Refresh
            </Button>
            <Button icon={<LogoutOutlined />} onClick={logout}>
              Log out
            </Button>
          </Space>
        </header>

        <section className="studio-grid">
          <Card className="studio-stat-accent">
            <Statistic
              title="Next set"
              value={nextSet ? formatDate(nextSet.startsAt) : 'None scheduled'}
              prefix={<CalendarOutlined />}
            />
          </Card>
          <Card className="studio-stat-accent">
            <Statistic
              title="Active stream keys"
              value={activeKeys.length}
              prefix={<KeyOutlined />}
            />
          </Card>
          <Card className="studio-stat-accent">
            <Statistic
              title="Live sessions"
              value={liveEvents.length}
              prefix={<PlayCircleOutlined />}
            />
          </Card>
        </section>

        <section className="studio-section-grid">
          <Card className="studio-section-card" title="Schedule">
            {schedule.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No scheduled sets" />
            ) : (
              <Table
                dataSource={schedule}
                rowKey="id"
                pagination={false}
                columns={[
                  {
                    title: 'Set',
                    dataIndex: 'title',
                    key: 'title',
                    render: (title, record) => (
                      <Space direction="vertical" size={0}>
                        <Text strong>{title}</Text>
                        <Text className="studio-muted">
                          {record.description || record.timezone}
                        </Text>
                      </Space>
                    ),
                  },
                  {
                    title: 'Starts',
                    dataIndex: 'startsAt',
                    key: 'startsAt',
                    render: formatDate,
                  },
                  {
                    title: 'Status',
                    dataIndex: 'status',
                    key: 'status',
                    render: value => <Tag color={statusColor(value)}>{value}</Tag>,
                  },
                  {
                    title: 'Alerts',
                    key: 'alerts',
                    render: (_, record) => (
                      <Space>
                        {record.notifyPush && <Tag color="cyan">push</Tag>}
                        {record.notifyEmail && <Tag color="green">email</Tag>}
                        {record.notifyWebhook && <Tag color="purple">webhook</Tag>}
                      </Space>
                    ),
                  },
                ]}
              />
            )}
          </Card>

          <Card
            className="studio-section-card"
            title="Stream keys"
            extra={
              <Button size="small" icon={<KeyOutlined />} onClick={() => setKeyModalOpen(true)}>
                New key
              </Button>
            }
          >
            <Table
              dataSource={streamKeys}
              rowKey="id"
              pagination={false}
              columns={[
                {
                  title: 'Label',
                  dataIndex: 'label',
                  key: 'label',
                  render: value => value || 'OBS key',
                },
                {
                  title: 'Last used',
                  dataIndex: 'lastUsedAt',
                  key: 'lastUsedAt',
                  render: value => (value ? formatDate(value) : 'Never'),
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
                  title: '',
                  key: 'action',
                  render: (_, key) =>
                    key.enabled && !key.revokedAt ? (
                      <Button
                        size="small"
                        danger
                        loading={saving}
                        onClick={() => revokeStreamKey(key.id)}
                      >
                        Revoke
                      </Button>
                    ) : null,
                },
              ]}
            />
          </Card>
        </section>

        <Card className="studio-section-card" title="Live history">
          <Table
            dataSource={liveEvents}
            rowKey="id"
            pagination={false}
            columns={[
              {
                title: 'Set',
                dataIndex: 'scheduleTitle',
                key: 'scheduleTitle',
                render: value => value || 'Unscheduled',
              },
              {
                title: 'Started',
                dataIndex: 'startedAt',
                key: 'startedAt',
                render: formatDate,
              },
              {
                title: 'Status',
                dataIndex: 'status',
                key: 'status',
                render: value => <Tag color={statusColor(value)}>{value}</Tag>,
              },
              {
                title: 'Alerts',
                dataIndex: 'notificationState',
                key: 'notificationState',
                render: value => <Tag color={statusColor(value)}>{value}</Tag>,
              },
            ]}
          />
        </Card>

        <Modal
          title="New stream key"
          open={keyModalOpen}
          onCancel={() => {
            setKeyModalOpen(false);
            setNewStreamKey('');
          }}
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
            <Form form={keyForm} layout="vertical" initialValues={{ label: 'Studio OBS key' }}>
              <Form.Item name="label" label="Label">
                <Input maxLength={80} placeholder="OBS laptop key" />
              </Form.Item>
            </Form>
          )}
        </Modal>
      </div>
    </main>
  );
}
