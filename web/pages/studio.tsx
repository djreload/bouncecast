import React, { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Empty,
  Form,
  Input,
  Modal,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  BOUNCECAST_STUDIO_LIVE_EVENTS,
  BOUNCECAST_STUDIO_LOGIN,
  BOUNCECAST_STUDIO_LOGOUT,
  BOUNCECAST_STUDIO_ME,
  BOUNCECAST_STUDIO_REGISTER,
  BOUNCECAST_STUDIO_SCHEDULE,
  BOUNCECAST_STUDIO_SCHEDULE_CANCEL,
  BOUNCECAST_STUDIO_SCHEDULE_UPDATE,
  BOUNCECAST_STUDIO_STREAM_KEYS,
  BOUNCECAST_STUDIO_STREAM_KEY_REVOKE,
  fetchStudioData,
} from '../utils/apis';

const CalendarOutlined = dynamic(() => import('@ant-design/icons/CalendarOutlined'), {
  ssr: false,
});
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });
const EditOutlined = dynamic(() => import('@ant-design/icons/EditOutlined'), { ssr: false });
const KeyOutlined = dynamic(() => import('@ant-design/icons/KeyOutlined'), { ssr: false });
const LogoutOutlined = dynamic(() => import('@ant-design/icons/LogoutOutlined'), { ssr: false });
const PlayCircleOutlined = dynamic(() => import('@ant-design/icons/PlayCircleOutlined'), {
  ssr: false,
});
const PlusOutlined = dynamic(() => import('@ant-design/icons/PlusOutlined'), { ssr: false });
const ReloadOutlined = dynamic(() => import('@ant-design/icons/ReloadOutlined'), { ssr: false });
const StopOutlined = dynamic(() => import('@ant-design/icons/StopOutlined'), { ssr: false });
const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });

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

type ScheduleFormValues = {
  title: string;
  description?: string;
  startsAt: string;
  endsAt?: string;
  timezone?: string;
  notifyEmail?: boolean;
  notifyPush?: boolean;
  notifyWebhook?: boolean;
};

function formatDate(value?: string) {
  if (!value) {
    return 'Not set';
  }
  return new Date(value).toLocaleString();
}

function formatDateTimeLocal(date: Date) {
  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return localDate.toISOString().slice(0, 16);
}

function toDateTimeLocal(value?: string) {
  if (!value) {
    return '';
  }
  return formatDateTimeLocal(new Date(value));
}

function defaultScheduleStart() {
  const date = new Date();
  date.setHours(date.getHours() + 1);
  date.setMinutes(0, 0, 0);
  return formatDateTimeLocal(date);
}

function getLocalTimezone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
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
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<ScheduleItem | null>(null);
  const [newStreamKey, setNewStreamKey] = useState('');
  const [registrationSubmitted, setRegistrationSubmitted] = useState(false);
  const [loginForm] = Form.useForm();
  const [registerForm] = Form.useForm();
  const [keyForm] = Form.useForm();
  const [scheduleForm] = Form.useForm<ScheduleFormValues>();

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

  const register = async () => {
    const values = await registerForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchStudioData(BOUNCECAST_STUDIO_REGISTER, undefined, {
        method: 'POST',
        data: values,
      });
      registerForm.resetFields();
      setRegistrationSubmitted(true);
      message.success(result.message || 'Registration received');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to register');
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

  const openScheduleModal = (item?: ScheduleItem) => {
    setEditingSchedule(item || null);
    scheduleForm.setFieldsValue({
      title: item?.title || '',
      description: item?.description || '',
      startsAt: item ? toDateTimeLocal(item.startsAt) : defaultScheduleStart(),
      endsAt: item ? toDateTimeLocal(item.endsAt) : '',
      timezone: item?.timezone || getLocalTimezone(),
      notifyEmail: item?.notifyEmail ?? false,
      notifyPush: item?.notifyPush ?? true,
      notifyWebhook: item?.notifyWebhook ?? true,
    });
    setScheduleModalOpen(true);
  };

  const saveScheduleItem = async () => {
    const values = await scheduleForm.validateFields();
    const payload = {
      ...values,
      startsAt: new Date(values.startsAt).toISOString(),
      endsAt: values.endsAt ? new Date(values.endsAt).toISOString() : '',
      timezone: values.timezone || getLocalTimezone(),
      notifyEmail: Boolean(values.notifyEmail),
      notifyPush: Boolean(values.notifyPush),
      notifyWebhook: Boolean(values.notifyWebhook),
    };

    setSaving(true);
    try {
      await fetchStudioData(
        editingSchedule ? BOUNCECAST_STUDIO_SCHEDULE_UPDATE : BOUNCECAST_STUDIO_SCHEDULE,
        token,
        {
          method: 'POST',
          data: editingSchedule ? { ...payload, id: editingSchedule.id } : payload,
        },
      );
      setScheduleModalOpen(false);
      setEditingSchedule(null);
      scheduleForm.resetFields();
      await loadStudio(token);
      message.success(editingSchedule ? 'Set updated' : 'Set scheduled');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save scheduled set');
    } finally {
      setSaving(false);
    }
  };

  const cancelScheduleItem = (id: number) => {
    Modal.confirm({
      title: 'Cancel scheduled set?',
      okText: 'Cancel set',
      okButtonProps: { danger: true },
      onOk: async () => {
        setSaving(true);
        try {
          await fetchStudioData(BOUNCECAST_STUDIO_SCHEDULE_CANCEL, token, {
            method: 'POST',
            data: { id },
          });
          await loadStudio(token);
          message.success('Set cancelled');
        } catch (error) {
          message.error(error instanceof Error ? error.message : 'Unable to cancel scheduled set');
        } finally {
          setSaving(false);
        }
      },
    });
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
              {registrationSubmitted && (
                <Alert
                  className="studio-register-alert"
                  type="info"
                  showIcon
                  message="Registration pending"
                  description="Your DJ account is inactive until an admin activates it."
                />
              )}
              <Tabs
                defaultActiveKey="login"
                items={[
                  {
                    key: 'login',
                    label: 'Log in',
                    children: (
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
                    ),
                  },
                  {
                    key: 'register',
                    label: 'Register',
                    children: (
                      <Form form={registerForm} layout="vertical">
                        <Form.Item
                          name="displayName"
                          label="DJ name"
                          rules={[{ required: true, message: 'Add your DJ name' }]}
                        >
                          <Input maxLength={80} autoComplete="name" placeholder="DJ name" />
                        </Form.Item>
                        <Form.Item
                          name="handle"
                          label="Handle"
                          rules={[{ required: true, message: 'Choose a handle' }]}
                        >
                          <Input
                            maxLength={32}
                            autoComplete="username"
                            placeholder="dj-name"
                            prefix="@"
                          />
                        </Form.Item>
                        <Form.Item
                          name="email"
                          label="Email"
                          rules={[
                            { required: true, message: 'Add your email' },
                            { type: 'email', message: 'Use a valid email address' },
                          ]}
                        >
                          <Input autoComplete="email" placeholder="dj@example.com" />
                        </Form.Item>
                        <Form.Item
                          name="password"
                          label="Password"
                          rules={[
                            { required: true, message: 'Choose a password' },
                            { min: 8, message: 'Use at least 8 characters' },
                          ]}
                        >
                          <Input.Password autoComplete="new-password" />
                        </Form.Item>
                        <Button
                          type="primary"
                          block
                          icon={<UserAddOutlined />}
                          loading={saving}
                          onClick={register}
                        >
                          Request DJ access
                        </Button>
                      </Form>
                    ),
                  },
                ]}
              />
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
          <Card
            className="studio-section-card"
            title="Schedule"
            extra={
              <Button
                size="small"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openScheduleModal()}
              >
                New set
              </Button>
            }
          >
            {schedule.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No scheduled sets" />
            ) : (
              <Table
                dataSource={schedule}
                rowKey="id"
                pagination={false}
                scroll={{ x: true }}
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
                  {
                    title: '',
                    key: 'actions',
                    render: (_, record) =>
                      record.status === 'planned' ? (
                        <Space>
                          <Button
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => openScheduleModal(record)}
                          >
                            Edit
                          </Button>
                          <Button
                            size="small"
                            danger
                            icon={<StopOutlined />}
                            loading={saving}
                            onClick={() => cancelScheduleItem(record.id)}
                          >
                            Cancel
                          </Button>
                        </Space>
                      ) : null,
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
              scroll={{ x: true }}
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
            scroll={{ x: true }}
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

        <Modal
          title={editingSchedule ? 'Edit scheduled set' : 'New scheduled set'}
          open={scheduleModalOpen}
          onCancel={() => {
            setScheduleModalOpen(false);
            setEditingSchedule(null);
            scheduleForm.resetFields();
          }}
          onOk={saveScheduleItem}
          confirmLoading={saving}
          okText={editingSchedule ? 'Save changes' : 'Schedule set'}
        >
          <Form form={scheduleForm} layout="vertical">
            <Form.Item
              name="title"
              label="Set title"
              rules={[{ required: true, message: 'Enter a set title' }]}
            >
              <Input maxLength={120} placeholder="Friday night live mix" />
            </Form.Item>
            <Form.Item name="description" label="Description">
              <Input.TextArea rows={3} maxLength={2000} placeholder="Genre, guests, or set notes" />
            </Form.Item>
            <div className="studio-form-grid">
              <Form.Item
                name="startsAt"
                label="Starts"
                rules={[{ required: true, message: 'Choose a start time' }]}
              >
                <Input type="datetime-local" />
              </Form.Item>
              <Form.Item name="endsAt" label="Ends">
                <Input type="datetime-local" />
              </Form.Item>
            </div>
            <Form.Item name="timezone" label="Timezone">
              <Input maxLength={64} placeholder="Europe/London" />
            </Form.Item>
            <Space direction="vertical" className="studio-checkbox-stack">
              <Form.Item name="notifyPush" valuePropName="checked">
                <Checkbox>Push alerts</Checkbox>
              </Form.Item>
              <Form.Item name="notifyEmail" valuePropName="checked">
                <Checkbox>Email alerts</Checkbox>
              </Form.Item>
              <Form.Item name="notifyWebhook" valuePropName="checked">
                <Checkbox>Webhook alerts</Checkbox>
              </Form.Item>
            </Space>
          </Form>
        </Modal>
      </div>
    </main>
  );
}
