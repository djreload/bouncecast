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
  BOUNCECAST_LIVE_EVENTS,
  BOUNCECAST_SCHEDULE,
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
    title: 'Alerts',
    key: 'alerts',
    render: (_, record) =>
      ['notifyPush', 'notifyEmail', 'notifyWebhook']
        .filter(field => record[field])
        .map(field => field.replace('notify', ''))
        .join(', ') || 'None',
  },
];

export default function Schedule() {
  const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
  const [streamers, setStreamers] = useState<Streamer[]>([]);
  const [liveEvents, setLiveEvents] = useState<GoLiveEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const loadStudioData = async () => {
    setLoading(true);
    try {
      const [scheduleResult, streamerResult, liveEventsResult] = await Promise.all([
        fetchData(BOUNCECAST_SCHEDULE),
        fetchData(BOUNCECAST_STREAMERS),
        fetchData(BOUNCECAST_LIVE_EVENTS),
      ]);
      setSchedule(scheduleResult || []);
      setStreamers(streamerResult || []);
      setLiveEvents(liveEventsResult || []);
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
                Push, email, webhook, and federation alerts are queued
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
    </div>
  );
}

Schedule.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
