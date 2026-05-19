import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import dynamic from 'next/dynamic';
import { AdminLayout } from '../../../components/layouts/AdminLayout';
import {
  BOUNCECAST_ADMIN_SESSION,
  BOUNCECAST_FACEBOOK_MESSENGER_ALERTS,
  BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_PREVIEW,
  BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_SETTINGS,
  BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_TEST,
  BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_VALIDATE,
  fetchData,
} from '../../../utils/apis';

const MessageOutlined = dynamic(() => import('@ant-design/icons/MessageOutlined'), { ssr: false });
const CheckCircleOutlined = dynamic(() => import('@ant-design/icons/CheckCircleOutlined'), {
  ssr: false,
});
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });

const { Title, Text, Paragraph } = Typography;

type MessengerSettings = {
  enabled: boolean;
  appId: string;
  appSecretSet: boolean;
  pageId: string;
  pageAccessTokenSet: boolean;
  webhookVerifyTokenSet: boolean;
  validateAppSecret: boolean;
  graphApiVersion: string;
  messageTemplate: string;
  liveUrlOverride: string;
  buttonLabel: string;
  sendDelaySeconds: number;
  cooldownSeconds: number;
  testRecipientPsid: string;
  lastSuccessfulSendAt: string;
  lastError: string;
  pagePostFallbackEnabled: boolean;
  pagePostTemplate: string;
  webhookCallbackPath: string;
  optInKeyword: string;
};

type Subscriber = {
  id: number;
  psid: string;
  displayName: string;
  source: string;
  optedIn: boolean;
  status: string;
  lastInteractionAt?: string;
  lastSentAt?: string;
  sendCount: number;
  failureCount: number;
};

type Campaign = {
  id: number;
  triggerType: string;
  status: string;
  attemptedCount: number;
  sentCount: number;
  skippedCount: number;
  failedCount: number;
  errorSummary: string;
  createdAt: string;
};

type AdminSession = {
  owner: boolean;
  admin: boolean;
  role: string;
};

type MessengerAlertsResponse = {
  settings: MessengerSettings;
  stats: {
    total: number;
    active: number;
    optedOut: number;
    failedBlocked: number;
  };
  subscribers: Subscriber[];
  campaigns: Campaign[];
};

function statusColor(status: string) {
  if (status === 'sent' || status === 'active') return 'green';
  if (status === 'failed' || status === 'blocked') return 'red';
  if (status === 'partial') return 'orange';
  if (status === 'opted_out' || status === 'skipped') return 'default';
  return 'blue';
}

export default function FacebookMessengerAlerts() {
  const [data, setData] = useState<MessengerAlertsResponse | null>(null);
  const [adminSession, setAdminSession] = useState<AdminSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState('');
  const [preview, setPreview] = useState('');
  const [form] = Form.useForm();

  const webhookURL = useMemo(() => {
    if (typeof window === 'undefined') return '';
    return `${window.location.origin}/integrations/facebook/messenger/webhook`;
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [messengerResult, sessionResult] = await Promise.all([
        fetchData(BOUNCECAST_FACEBOOK_MESSENGER_ALERTS),
        fetchData(BOUNCECAST_ADMIN_SESSION),
      ]);
      setData(messengerResult);
      setAdminSession(sessionResult || null);
      form.setFieldsValue(messengerResult.settings || {});
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const canManage = Boolean(adminSession?.owner);

  const saveSettings = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await fetchData(BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_SETTINGS, {
        method: 'POST',
        data: values,
      });
      setNotice('Facebook Messenger alert settings saved.');
      await loadData();
    } finally {
      setSaving(false);
    }
  };

  const validateConfig = async () => {
    const result = await fetchData(BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_VALIDATE, {
      method: 'POST',
      data: {},
    });
    setNotice(`Meta config validated for Page ${result.page?.name || result.page?.id || ''}.`);
  };

  const sendTest = async (goLiveStyle: boolean) => {
    const psid = form.getFieldValue('testRecipientPsid');
    await fetchData(BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_TEST, {
      method: 'POST',
      data: { psid, goLiveStyle },
    });
    setNotice(goLiveStyle ? 'Test go-live alert sent.' : 'Test message sent.');
    await loadData();
  };

  const loadPreview = async () => {
    const result = await fetchData(BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_PREVIEW, {
      method: 'POST',
      data: {},
    });
    setPreview(result.message || '');
  };

  const copyText = async (value: string, label: string) => {
    await navigator.clipboard?.writeText(value);
    setNotice(`${label} copied.`);
  };

  const settings = data?.settings;

  return (
    <div className="bouncecast-admin-page messenger-alerts-page">
      <div className="studio-hero">
        <div>
          <Text className="studio-eyebrow">Integrations</Text>
          <Title level={1}>Facebook Messenger Alerts</Title>
          <Text>
            Send Meta-policy-safe Messenger go-live alerts to opted-in Page message subscribers.
          </Text>
        </div>
        <Space wrap>
          <Button icon={<CopyOutlined />} onClick={() => copyText(webhookURL, 'Webhook URL')}>
            Copy webhook URL
          </Button>
          <Button icon={<CheckCircleOutlined />} disabled={!canManage} onClick={validateConfig}>
            Validate Meta config
          </Button>
        </Space>
      </div>

      {notice && <Alert className="studio-panel" type="success" showIcon message={notice} />}
      {settings?.lastError && (
        <Alert className="studio-panel" type="warning" showIcon message={settings.lastError} />
      )}

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={6}>
          <Card>
            <Statistic
              title="Subscribers"
              value={data?.stats.total || 0}
              prefix={<MessageOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card>
            <Statistic title="Active" value={data?.stats.active || 0} />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card>
            <Statistic title="Opted out" value={data?.stats.optedOut || 0} />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card>
            <Statistic title="Failed/blocked" value={data?.stats.failedBlocked || 0} />
          </Card>
        </Col>
      </Row>

      <Card className="studio-panel" title="Meta Messenger setup">
        <Alert
          type="info"
          showIcon
          message="Messenger alerts only go to users who message the Page or use an allowed opt-in flow. BounceCast does not message all Page followers."
          className="studio-panel"
        />
        <Form form={form} layout="vertical" disabled={!canManage} initialValues={settings || {}}>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name="enabled" label="Enable Messenger alerts" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="validateAppSecret"
                label="Validate webhook app secret"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="pagePostFallbackEnabled"
                label="Page post fallback"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name="appId" label="Facebook App ID">
                <Input maxLength={120} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name="pageId" label="Facebook Page ID">
                <Input maxLength={120} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name="graphApiVersion" label="Graph API version">
                <Input placeholder="v25.0" maxLength={12} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item
                name="appSecret"
                label={`Facebook App Secret${settings?.appSecretSet ? ' (stored)' : ''}`}
              >
                <Input.Password
                  placeholder={settings?.appSecretSet ? 'Stored secret is masked' : ''}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="pageAccessToken"
                label={`Page Access Token${settings?.pageAccessTokenSet ? ' (stored)' : ''}`}
              >
                <Input.Password
                  placeholder={settings?.pageAccessTokenSet ? 'Stored token is masked' : ''}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="webhookVerifyToken"
                label={`Webhook Verify Token${settings?.webhookVerifyTokenSet ? ' (stored)' : ''}`}
              >
                <Input.Password
                  placeholder={
                    settings?.webhookVerifyTokenSet ? 'Stored verify token is masked' : ''
                  }
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name="buttonLabel" label="Button label">
                <Input maxLength={20} placeholder="Watch Live" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name="sendDelaySeconds" label="Send delay after go-live">
                <InputNumber min={0} max={900} addonAfter="seconds" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name="cooldownSeconds" label="Cooldown between sends">
                <InputNumber min={0} max={86400} addonAfter="seconds" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="liveUrlOverride" label="Default live URL override">
            <Input placeholder="Leave empty to use the configured BounceCast server URL" />
          </Form.Item>
          <Form.Item name="messageTemplate" label="Default live alert message template">
            <Input.TextArea rows={5} />
          </Form.Item>
          <Form.Item name="pagePostTemplate" label="Facebook Page post fallback template">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="testRecipientPsid" label="Test recipient PSID">
            <Input maxLength={120} />
          </Form.Item>
          <Space wrap>
            <Button type="primary" loading={saving} onClick={saveSettings}>
              Save settings
            </Button>
            <Button onClick={() => sendTest(false)}>Send test message</Button>
            <Button onClick={() => sendTest(true)}>Send test go-live alert</Button>
            <Button onClick={loadPreview}>Preview message</Button>
          </Space>
        </Form>
        {preview && (
          <Paragraph
            className="studio-secret-box"
            style={{ marginTop: 16, whiteSpace: 'pre-wrap' }}
          >
            {preview}
          </Paragraph>
        )}
      </Card>

      <Card className="studio-panel" title="Webhook setup">
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text>Callback URL</Text>
          <Input
            readOnly
            value={webhookURL}
            addonAfter={
              <Button type="link" onClick={() => copyText(webhookURL, 'Webhook URL')}>
                Copy
              </Button>
            }
          />
          <Text type="secondary">
            Use the verify token you entered above in Meta. Stored verify tokens stay masked in
            BounceCast.
          </Text>
        </Space>
      </Card>

      <Card className="studio-panel" title="Subscribers">
        <Table
          loading={loading}
          rowKey="id"
          dataSource={data?.subscribers || []}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: 'PSID', dataIndex: 'psid', key: 'psid' },
            { title: 'Source', dataIndex: 'source', key: 'source' },
            {
              title: 'Status',
              dataIndex: 'status',
              key: 'status',
              render: status => <Tag color={statusColor(status)}>{status}</Tag>,
            },
            { title: 'Sends', dataIndex: 'sendCount', key: 'sendCount' },
            { title: 'Failures', dataIndex: 'failureCount', key: 'failureCount' },
            {
              title: 'Last interaction',
              dataIndex: 'lastInteractionAt',
              key: 'lastInteractionAt',
              render: value => (value ? new Date(value).toLocaleString() : 'Never'),
            },
          ]}
        />
      </Card>

      <Card className="studio-panel" title="Campaign logs">
        <Table
          loading={loading}
          rowKey="id"
          dataSource={data?.campaigns || []}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: 'ID', dataIndex: 'id', key: 'id' },
            { title: 'Trigger', dataIndex: 'triggerType', key: 'triggerType' },
            {
              title: 'Status',
              dataIndex: 'status',
              key: 'status',
              render: status => <Tag color={statusColor(status)}>{status}</Tag>,
            },
            { title: 'Attempted', dataIndex: 'attemptedCount', key: 'attemptedCount' },
            { title: 'Sent', dataIndex: 'sentCount', key: 'sentCount' },
            { title: 'Skipped', dataIndex: 'skippedCount', key: 'skippedCount' },
            { title: 'Failed', dataIndex: 'failedCount', key: 'failedCount' },
            { title: 'Error', dataIndex: 'errorSummary', key: 'errorSummary' },
            {
              title: 'Created',
              dataIndex: 'createdAt',
              key: 'createdAt',
              render: value => (value ? new Date(value).toLocaleString() : ''),
            },
          ]}
        />
      </Card>
    </div>
  );
}

FacebookMessengerAlerts.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
