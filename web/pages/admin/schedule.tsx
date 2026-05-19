import React, { ReactElement, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Button,
  Card,
  Checkbox,
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
  Timeline,
  Typography,
} from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';
import {
  BOUNCECAST_EMAIL_SETTINGS,
  BOUNCECAST_ADMIN_SESSION,
  BOUNCECAST_MESSENGER_SETTINGS,
  BOUNCECAST_LIVE_EVENTS,
  BOUNCECAST_NOTIFICATION_DELIVERIES,
  BOUNCECAST_NOTIFICATION_DELIVERIES_EXPORT,
  BOUNCECAST_NOTIFICATION_DELIVERIES_RETRY_FAILED,
  BOUNCECAST_NOTIFICATION_SUBSCRIBERS,
  BOUNCECAST_NOTIFICATION_SUBSCRIBER_DISABLE,
  BOUNCECAST_PUSH_SETTINGS,
  BOUNCECAST_SCHEDULE,
  BOUNCECAST_SCHEDULE_REMINDER_DISABLE,
  BOUNCECAST_SCHEDULE_REMINDERS,
  BOUNCECAST_STREAMERS,
  fetchData,
} from '../../utils/apis';

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
const ReloadOutlined = dynamic(() => import('@ant-design/icons/ReloadOutlined'), { ssr: false });
const DownloadOutlined = dynamic(() => import('@ant-design/icons/DownloadOutlined'), {
  ssr: false,
});

const { Title, Text } = Typography;

type Streamer = {
  id: number;
  displayName: string;
};

type ScheduleItem = {
  id: number;
  title: string;
  streamer: string;
  startsAt: string;
  status: string;
  visibility: string;
  notifyEmail: boolean;
  notifyPush: boolean;
  notifyWebhook: boolean;
};

type GoLiveEvent = {
  id: number;
  streamer: string;
  scheduleTitle: string;
  startedAt: string;
  endedAt?: string;
  status: string;
  notificationState: string;
};

type NotificationSubscriber = {
  id: number;
  channel: string;
  destination: string;
  displayName: string;
  createdAt: string;
  disabledAt?: string;
};

type NotificationDelivery = {
  id: number;
  streamer: string;
  channel: string;
  destination: string;
  status: string;
  attemptCount: number;
  lastError: string;
  createdAt: string;
  sentAt?: string;
};

type EmailSettings = {
  enabled: boolean;
  provider: string;
  host: string;
  port: number;
  username: string;
  passwordSet: boolean;
  fromAddress: string;
  fromName: string;
  startTls: boolean;
  subject: string;
};

type PushSettings = {
  enabled: boolean;
  subscriberCount: number;
  publicKeySet: boolean;
  privateKeySet: boolean;
  goLiveMessage: string;
};

type MessengerSettings = {
  enabled: boolean;
  graphApiVersion: string;
  pageAccessTokenSet: boolean;
  messageTemplate: string;
};

type ScheduleReminder = {
  id: number;
  scheduleId: number;
  scheduleTitle: string;
  streamer: string;
  displayName: string;
  email: string;
  notifyEmail: boolean;
  notifyPush: boolean;
  notifyMessenger: boolean;
  messengerDestination: string;
  browserPushLinked: boolean;
  lastQueuedAt?: string;
  lastDeliveryStatus: string;
  lastDeliveryError: string;
  disabledAt?: string;
  updatedAt: string;
};

type AdminSession = {
  role: string;
  owner: boolean;
  admin: boolean;
};

const columns = [
  {
    title: 'Set',
    dataIndex: 'title',
    key: 'title',
    render: (title, record) => (
      <Space direction="vertical" size={0}>
        <Text strong>{title}</Text>
        <Text type="secondary">{record.streamer || 'Unassigned'}</Text>
      </Space>
    ),
  },
  {
    title: 'Start',
    dataIndex: 'startsAt',
    key: 'start',
    render: startsAt => new Date(startsAt).toLocaleString(),
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    render: status => <Tag color={status === 'planned' ? 'purple' : 'default'}>{status}</Tag>,
  },
  {
    title: 'Visibility',
    dataIndex: 'visibility',
    key: 'visibility',
    render: visibility => (
      <Tag color={visibility === 'public' ? 'green' : 'default'}>{visibility}</Tag>
    ),
  },
  {
    title: 'Alerts',
    key: 'alerts',
    render: (_, record) =>
      ['notifyPush', 'notifyEmail', 'notifyWebhook']
        .filter(field => record[field])
        .map(field => field.replace('notify', ''))
        .join(', ') || 'None',
  },
];

function deliveryStatusColor(status: string) {
  if (status === 'sent') {
    return 'green';
  }
  if (status === 'failed') {
    return 'red';
  }
  return 'gold';
}

export default function Schedule() {
  const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
  const [streamers, setStreamers] = useState<Streamer[]>([]);
  const [liveEvents, setLiveEvents] = useState<GoLiveEvent[]>([]);
  const [subscribers, setSubscribers] = useState<NotificationSubscriber[]>([]);
  const [deliveries, setDeliveries] = useState<NotificationDelivery[]>([]);
  const [reminders, setReminders] = useState<ScheduleReminder[]>([]);
  const [emailSettings, setEmailSettings] = useState<EmailSettings | null>(null);
  const [pushSettings, setPushSettings] = useState<PushSettings | null>(null);
  const [messengerSettings, setMessengerSettings] = useState<MessengerSettings | null>(null);
  const [adminSession, setAdminSession] = useState<AdminSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [subscriberModalOpen, setSubscriberModalOpen] = useState(false);
  const [emailSettingsModalOpen, setEmailSettingsModalOpen] = useState(false);
  const [messengerSettingsModalOpen, setMessengerSettingsModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [subscriberForm] = Form.useForm();
  const [emailSettingsForm] = Form.useForm();
  const [messengerSettingsForm] = Form.useForm();

  const loadStudioData = async () => {
    setLoading(true);
    try {
      const [
        scheduleResult,
        streamerResult,
        liveEventsResult,
        subscriberResult,
        deliveryResult,
        reminderResult,
        emailSettingsResult,
        pushSettingsResult,
        messengerSettingsResult,
        adminSessionResult,
      ] = await Promise.all([
        fetchData(BOUNCECAST_SCHEDULE),
        fetchData(BOUNCECAST_STREAMERS),
        fetchData(BOUNCECAST_LIVE_EVENTS),
        fetchData(BOUNCECAST_NOTIFICATION_SUBSCRIBERS),
        fetchData(BOUNCECAST_NOTIFICATION_DELIVERIES),
        fetchData(BOUNCECAST_SCHEDULE_REMINDERS),
        fetchData(BOUNCECAST_EMAIL_SETTINGS),
        fetchData(BOUNCECAST_PUSH_SETTINGS),
        fetchData(BOUNCECAST_MESSENGER_SETTINGS),
        fetchData(BOUNCECAST_ADMIN_SESSION),
      ]);
      setSchedule(scheduleResult || []);
      setStreamers(streamerResult || []);
      setLiveEvents(liveEventsResult || []);
      setSubscribers(subscriberResult || []);
      setDeliveries(deliveryResult || []);
      setReminders(reminderResult || []);
      setEmailSettings(emailSettingsResult || null);
      setPushSettings(pushSettingsResult || null);
      setMessengerSettings(messengerSettingsResult || null);
      setAdminSession(adminSessionResult || null);
      emailSettingsForm.setFieldsValue(emailSettingsResult || {});
      messengerSettingsForm.setFieldsValue(messengerSettingsResult || {});
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStudioData();
  }, []);

  const createScheduleItem = async () => {
    const values = await form.validateFields();
    const startsAt = new Date(values.startsAt).toISOString();
    const endsAt = values.endsAt ? new Date(values.endsAt).toISOString() : '';
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_SCHEDULE, {
        method: 'POST',
        data: {
          ...values,
          startsAt,
          endsAt,
        },
      });
      form.resetFields();
      setModalOpen(false);
      await loadStudioData();
    } finally {
      setSaving(false);
    }
  };

  const createSubscriber = async () => {
    const values = await subscriberForm.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_NOTIFICATION_SUBSCRIBERS, {
        method: 'POST',
        data: values,
      });
      subscriberForm.resetFields();
      setSubscriberModalOpen(false);
      await loadStudioData();
    } finally {
      setSaving(false);
    }
  };

  const disableSubscriber = async (id: number) => {
    await fetchData(BOUNCECAST_NOTIFICATION_SUBSCRIBER_DISABLE, {
      method: 'POST',
      data: { id },
    });
    await loadStudioData();
  };

  const retryFailedDeliveries = async () => {
    await fetchData(BOUNCECAST_NOTIFICATION_DELIVERIES_RETRY_FAILED, {
      method: 'POST',
      data: { channel: 'all' },
    });
    await loadStudioData();
  };

  const exportDeliveries = () => {
    window.open(BOUNCECAST_NOTIFICATION_DELIVERIES_EXPORT, '_blank', 'noopener,noreferrer');
  };

  const saveEmailSettings = async () => {
    const values = await emailSettingsForm.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_EMAIL_SETTINGS, {
        method: 'POST',
        data: {
          ...values,
          port: Number(values.port),
        },
      });
      setEmailSettingsModalOpen(false);
      emailSettingsForm.resetFields(['password']);
      await loadStudioData();
    } finally {
      setSaving(false);
    }
  };

  const saveMessengerSettings = async () => {
    const values = await messengerSettingsForm.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_MESSENGER_SETTINGS, {
        method: 'POST',
        data: values,
      });
      setMessengerSettingsModalOpen(false);
      messengerSettingsForm.resetFields(['pageAccessToken']);
      await loadStudioData();
    } finally {
      setSaving(false);
    }
  };

  const disableReminder = async (id: number) => {
    await fetchData(BOUNCECAST_SCHEDULE_REMINDER_DISABLE, {
      method: 'POST',
      data: { id },
    });
    await loadStudioData();
  };

  const applyEmailProviderPreset = (provider: string) => {
    if (provider === 'brevo') {
      emailSettingsForm.setFieldsValue({
        host: 'smtp-relay.brevo.com',
        port: 587,
        startTls: true,
      });
    }
  };

  const pushReady = Boolean(
    pushSettings?.enabled && pushSettings.publicKeySet && pushSettings.privateKeySet,
  );
  const messengerReady = Boolean(
    messengerSettings?.enabled && messengerSettings.pageAccessTokenSet,
  );
  const activeAutomationCount = [emailSettings?.enabled, pushReady, messengerReady].filter(
    Boolean,
  ).length;
  const canEditSensitiveDeliverySettings = Boolean(adminSession?.owner);
  const activeSubscriberCount =
    subscribers.filter(subscriber => !subscriber.disabledAt).length +
    (pushSettings?.subscriberCount || 0);

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
        <Button type="primary" icon={<CalendarOutlined />} onClick={() => setModalOpen(true)}>
          New set
        </Button>
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Upcoming sets"
              value={schedule.length}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Automation channels"
              value={activeAutomationCount}
              suffix="/ 3"
              prefix={<ThunderboltOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <Statistic
              title="Subscriber channels"
              value={activeSubscriberCount}
              prefix={<BellOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={15}>
          <Card title="Schedule" className="studio-panel">
            <Table
              columns={columns}
              dataSource={schedule}
              loading={loading}
              rowKey="id"
              pagination={false}
            />
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
                Browser push, email, webhook, and federation alerts are queued
              </Timeline.Item>
              <Timeline.Item color="magenta">
                Account reminders can target email, mapped browser push subscriptions, and Messenger
                contacts
              </Timeline.Item>
            </Timeline>
          </Card>
        </Col>
      </Row>

      <Card title="Recent live events" className="studio-panel">
        <Table
          dataSource={liveEvents}
          rowKey="id"
          pagination={false}
          columns={[
            {
              title: 'DJ',
              dataIndex: 'streamer',
              key: 'streamer',
              render: streamer => streamer || 'Unknown',
            },
            {
              title: 'Schedule',
              dataIndex: 'scheduleTitle',
              key: 'scheduleTitle',
              render: scheduleTitle => scheduleTitle || 'Unscheduled',
            },
            {
              title: 'Started',
              dataIndex: 'startedAt',
              key: 'startedAt',
              render: startedAt => new Date(startedAt).toLocaleString(),
            },
            {
              title: 'Status',
              dataIndex: 'status',
              key: 'status',
              render: status => <Tag color={status === 'live' ? 'green' : 'default'}>{status}</Tag>,
            },
            {
              title: 'Alerts',
              dataIndex: 'notificationState',
              key: 'notificationState',
            },
          ]}
        />
      </Card>

      <Card
        title="Notification subscribers"
        className="studio-panel"
        extra={
          <Space>
            <Tag color={pushReady ? 'green' : 'default'}>
              Browser push {pushReady ? 'ready' : 'off'}: {pushSettings?.subscriberCount || 0}
            </Tag>
            {canEditSensitiveDeliverySettings && (
              <>
                <Button size="small" onClick={() => setEmailSettingsModalOpen(true)}>
                  Email settings
                </Button>
                <Button size="small" onClick={() => setMessengerSettingsModalOpen(true)}>
                  Messenger settings
                </Button>
              </>
            )}
            <Button
              size="small"
              icon={<BellOutlined />}
              onClick={() => setSubscriberModalOpen(true)}
            >
              Add subscriber
            </Button>
          </Space>
        }
      >
        <Table
          dataSource={subscribers}
          rowKey="id"
          pagination={false}
          columns={[
            {
              title: 'Name',
              dataIndex: 'displayName',
              key: 'displayName',
              render: displayName => displayName || 'Subscriber',
            },
            {
              title: 'Channel',
              dataIndex: 'channel',
              key: 'channel',
              render: channel => <Tag>{channel}</Tag>,
            },
            {
              title: 'Destination',
              dataIndex: 'destination',
              key: 'destination',
            },
            {
              title: 'Status',
              key: 'status',
              render: (_, subscriber) => (
                <Tag color={subscriber.disabledAt ? 'default' : 'green'}>
                  {subscriber.disabledAt ? 'disabled' : 'active'}
                </Tag>
              ),
            },
            {
              title: 'Action',
              key: 'action',
              render: (_, subscriber) =>
                subscriber.disabledAt ? null : (
                  <Button size="small" danger onClick={() => disableSubscriber(subscriber.id)}>
                    Disable
                  </Button>
                ),
            },
          ]}
        />
      </Card>

      <Card title="Account schedule reminders" className="studio-panel">
        <Table
          dataSource={reminders}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          columns={[
            {
              title: 'Set',
              dataIndex: 'scheduleTitle',
              key: 'scheduleTitle',
              render: (title, record) => (
                <Space direction="vertical" size={0}>
                  <Text strong>{title}</Text>
                  <Text type="secondary">{record.streamer || 'Unassigned'}</Text>
                </Space>
              ),
            },
            {
              title: 'Viewer',
              key: 'viewer',
              render: (_, reminder) => (
                <Space direction="vertical" size={0}>
                  <Text>{reminder.displayName || 'Viewer'}</Text>
                  <Text type="secondary">
                    {reminder.email || reminder.messengerDestination || 'No direct destination'}
                  </Text>
                </Space>
              ),
            },
            {
              title: 'Channels',
              key: 'channels',
              render: (_, reminder) => (
                <Space wrap>
                  {reminder.notifyEmail && <Tag color="blue">email</Tag>}
                  {reminder.notifyPush && (
                    <Tag color={reminder.browserPushLinked ? 'green' : 'gold'}>browser push</Tag>
                  )}
                  {reminder.notifyMessenger && <Tag color="magenta">Messenger</Tag>}
                </Space>
              ),
            },
            {
              title: 'Delivery',
              key: 'delivery',
              render: (_, reminder) => (
                <Space direction="vertical" size={0}>
                  <Tag
                    color={
                      reminder.disabledAt
                        ? 'default'
                        : deliveryStatusColor(reminder.lastDeliveryStatus || 'queued')
                    }
                  >
                    {reminder.disabledAt ? 'disabled' : reminder.lastDeliveryStatus || 'scheduled'}
                  </Tag>
                  {reminder.lastDeliveryError && (
                    <Text type="secondary">{reminder.lastDeliveryError}</Text>
                  )}
                </Space>
              ),
            },
            {
              title: 'Action',
              key: 'action',
              render: (_, reminder) =>
                reminder.disabledAt ? null : (
                  <Button size="small" danger onClick={() => disableReminder(reminder.id)}>
                    Disable
                  </Button>
                ),
            },
          ]}
        />
      </Card>

      <Card
        title="Notification deliveries"
        className="studio-panel"
        extra={
          <Space>
            <Button size="small" icon={<ReloadOutlined />} onClick={retryFailedDeliveries}>
              Retry failed
            </Button>
            <Button size="small" icon={<DownloadOutlined />} onClick={exportDeliveries}>
              Export CSV
            </Button>
          </Space>
        }
      >
        <Table
          dataSource={deliveries}
          rowKey="id"
          pagination={false}
          columns={[
            {
              title: 'DJ',
              dataIndex: 'streamer',
              key: 'streamer',
              render: streamer => streamer || 'Unknown',
            },
            {
              title: 'Channel',
              dataIndex: 'channel',
              key: 'channel',
              render: channel => <Tag>{channel}</Tag>,
            },
            {
              title: 'Status',
              dataIndex: 'status',
              key: 'status',
              render: status => <Tag color={deliveryStatusColor(status)}>{status}</Tag>,
            },
            {
              title: 'Attempts',
              dataIndex: 'attemptCount',
              key: 'attemptCount',
            },
            {
              title: 'Created',
              dataIndex: 'createdAt',
              key: 'createdAt',
              render: createdAt => new Date(createdAt).toLocaleString(),
            },
            {
              title: 'Error',
              dataIndex: 'lastError',
              key: 'lastError',
              render: lastError => lastError || 'None',
            },
          ]}
        />
      </Card>

      {!loading && schedule.length === 0 && (
        <Card className="studio-panel">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="Create the first planned DJ set to begin testing the live calendar."
          />
        </Card>
      )}

      <Modal
        title="New DJ set"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={createScheduleItem}
        confirmLoading={saving}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
            status: 'planned',
            visibility: 'public',
            notifyWebhook: true,
          }}
        >
          <Form.Item name="streamerId" label="Streamer">
            <Select
              allowClear
              placeholder="Assign a DJ"
              options={streamers.map(streamer => ({
                label: streamer.displayName,
                value: streamer.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="title"
            label="Set title"
            rules={[{ required: true, message: 'Add a set title' }]}
          >
            <Input placeholder="Friday night house session" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} />
          </Form.Item>
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
          <Form.Item name="timezone" label="Timezone">
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="status" label="Status">
                <Select
                  options={[
                    { label: 'Planned', value: 'planned' },
                    { label: 'Live now', value: 'live' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="visibility" label="Visibility">
                <Select
                  options={[
                    { label: 'Public lineup', value: 'public' },
                    { label: 'Private/internal', value: 'private' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Space direction="vertical">
            <Form.Item name="notifyPush" valuePropName="checked" noStyle>
              <Checkbox>Push notification</Checkbox>
            </Form.Item>
            <Form.Item name="notifyEmail" valuePropName="checked" noStyle>
              <Checkbox>Email notification</Checkbox>
            </Form.Item>
            <Form.Item name="notifyWebhook" valuePropName="checked" noStyle>
              <Checkbox>Webhook notification</Checkbox>
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      <Modal
        title="Add notification subscriber"
        open={subscriberModalOpen}
        onCancel={() => setSubscriberModalOpen(false)}
        onOk={createSubscriber}
        confirmLoading={saving}
      >
        <Form
          form={subscriberForm}
          layout="vertical"
          initialValues={{
            channel: 'email',
          }}
        >
          <Form.Item name="displayName" label="Display name">
            <Input placeholder="Promoter list" />
          </Form.Item>
          <Form.Item
            name="channel"
            label="Channel"
            rules={[{ required: true, message: 'Choose a channel' }]}
          >
            <Select
              options={[
                { label: 'Email', value: 'email' },
                { label: 'Push', value: 'push' },
                { label: 'Webhook', value: 'webhook' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="destination"
            label="Destination"
            rules={[{ required: true, message: 'Add a destination' }]}
          >
            <Input placeholder="email@example.com or endpoint URL" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Email settings"
        open={emailSettingsModalOpen}
        onCancel={() => setEmailSettingsModalOpen(false)}
        onOk={saveEmailSettings}
        confirmLoading={saving}
      >
        <Form
          form={emailSettingsForm}
          layout="vertical"
          initialValues={{
            enabled: false,
            provider: 'custom',
            port: 587,
            fromName: 'BounceCast',
            startTls: true,
            subject: '{{streamer}} is live on BounceCast',
          }}
        >
          <Form.Item name="enabled" valuePropName="checked">
            <Checkbox>Enable SMTP email delivery</Checkbox>
          </Form.Item>
          <Form.Item name="provider" label="Provider">
            <Select
              onChange={applyEmailProviderPreset}
              options={[
                { label: 'Custom SMTP', value: 'custom' },
                { label: 'Brevo SMTP relay', value: 'brevo' },
              ]}
            />
          </Form.Item>
          <Row gutter={12}>
            <Col span={16}>
              <Form.Item name="host" label="SMTP host">
                <Input placeholder="smtp.example.com" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="port" label="Port">
                <Input type="number" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="username" label="Username">
            <Input placeholder="Brevo login email or SMTP username" />
          </Form.Item>
          <Form.Item
            name="password"
            label={emailSettings?.passwordSet ? 'Password (saved)' : 'Password'}
          >
            <Input.Password
              placeholder={
                emailSettings?.passwordSet
                  ? 'Leave blank to keep saved password'
                  : 'Brevo SMTP key or SMTP password'
              }
            />
          </Form.Item>
          <Form.Item name="fromAddress" label="From address">
            <Input placeholder="alerts@example.com" />
          </Form.Item>
          <Form.Item name="fromName" label="From name">
            <Input />
          </Form.Item>
          <Form.Item name="subject" label="Subject">
            <Input />
          </Form.Item>
          <Form.Item name="startTls" valuePropName="checked">
            <Checkbox>Use STARTTLS</Checkbox>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Messenger settings"
        open={messengerSettingsModalOpen}
        onCancel={() => setMessengerSettingsModalOpen(false)}
        onOk={saveMessengerSettings}
        confirmLoading={saving}
      >
        <Form
          form={messengerSettingsForm}
          layout="vertical"
          initialValues={{
            enabled: false,
            graphApiVersion: 'v20.0',
            messageTemplate: '{{streamer}} is live on BounceCast: {{schedule}}',
          }}
        >
          <Form.Item name="enabled" valuePropName="checked">
            <Checkbox>Enable Messenger schedule reminder delivery</Checkbox>
          </Form.Item>
          <Form.Item name="graphApiVersion" label="Graph API version">
            <Input placeholder="v20.0" />
          </Form.Item>
          <Form.Item
            name="pageAccessToken"
            label={
              messengerSettings?.pageAccessTokenSet
                ? 'Page access token (saved)'
                : 'Page access token'
            }
          >
            <Input.Password
              placeholder={
                messengerSettings?.pageAccessTokenSet
                  ? 'Leave blank to keep saved token'
                  : 'Facebook Page Access Token'
              }
            />
          </Form.Item>
          <Form.Item name="messageTemplate" label="Message template">
            <Input maxLength={240} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

Schedule.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
