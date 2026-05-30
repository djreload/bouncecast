import React, { ReactElement, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';
import {
  BOUNCECAST_STARS_ADMIN,
  BOUNCECAST_STARS_PACKAGES,
  BOUNCECAST_STARS_SETTINGS,
  BOUNCECAST_STARS_TEST_OVERLAY,
  BOUNCECAST_STARS_WALLET_ADJUST,
  fetchData,
} from '../../utils/apis';
import { StarAdminSummary, StarPackage, StarSettings } from '../../interfaces/stars.model';

const { Title, Text } = Typography;

const starEffectOptions = [
  { label: 'Sparkle', value: 'sparkle' },
  { label: 'Fireworks', value: 'fireworks' },
  { label: 'Hearts', value: 'hearts' },
  { label: 'Hype', value: 'hype' },
  { label: 'DJ drop', value: 'dj_drop' },
];

const defaultSettings: StarSettings = {
  enabled: false,
  paypalEnvironment: 'sandbox',
  paypalClientId: '',
  paypalClientSecret: '',
  paypalWebhookId: '',
  currency: 'GBP',
  supportMessage:
    'Stars are a fun way to support this site and trigger live on-screen effects. Stars have no cash value and are not paid out to streamers.',
  minimumSendAmount: 1,
  maximumSendAmount: 7000,
  sendCooldownSeconds: 10,
  overlayEffectsEnabled: true,
  soundEffectsEnabled: true,
  debugLoggingEnabled: false,
};

function formatPrice(record: StarPackage) {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: record.currency || 'GBP',
  }).format((record.priceCents || 0) / 100);
}

function rankTagColor(rank: number) {
  if (rank === 1) {
    return 'gold';
  }
  if (rank === 2) {
    return 'blue';
  }
  if (rank === 3) {
    return 'magenta';
  }
  return 'default';
}

export default function StarsAdmin() {
  const [summary, setSummary] = useState<StarAdminSummary>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingOverlay, setTestingOverlay] = useState(false);
  const [packageModalOpen, setPackageModalOpen] = useState(false);
  const [editingPackage, setEditingPackage] = useState<StarPackage>(null);
  const [settingsForm] = Form.useForm<StarSettings>();
  const [packageForm] = Form.useForm<StarPackage>();
  const [overlayTestForm] = Form.useForm();
  const [adjustForm] = Form.useForm();
  const paypalWebhookUrl =
    typeof window === 'undefined'
      ? '/api/stars/paypal/webhook'
      : `${window.location.origin}/api/stars/paypal/webhook`;

  const loadStars = async () => {
    setLoading(true);
    try {
      const result = await fetchData(BOUNCECAST_STARS_ADMIN);
      setSummary(result);
      settingsForm.setFieldsValue({ ...defaultSettings, ...result.settings });
      overlayTestForm.setFieldsValue({
        soundEnabled: result.settings?.soundEffectsEnabled ?? true,
      });
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load Stars');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStars();
  }, []);

  const saveSettings = async () => {
    setSaving(true);
    try {
      const values = await settingsForm.validateFields();
      await fetchData(BOUNCECAST_STARS_SETTINGS, { method: 'POST', data: values });
      message.success('Stars settings saved');
      loadStars();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save settings');
    } finally {
      setSaving(false);
    }
  };

  const openPackageModal = (pkg?: StarPackage) => {
    setEditingPackage(pkg || null);
    packageForm.setFieldsValue(
      pkg || {
        name: '',
        starAmount: 100,
        priceCents: 100,
        currency: summary?.settings?.currency || 'GBP',
        enabled: true,
        displayOrder: 100,
      },
    );
    setPackageModalOpen(true);
  };

  const savePackage = async () => {
    try {
      const values = await packageForm.validateFields();
      await fetchData(BOUNCECAST_STARS_PACKAGES, {
        method: 'POST',
        data: { ...values, id: editingPackage?.id || 0 },
      });
      message.success('Star package saved');
      setPackageModalOpen(false);
      loadStars();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save package');
    }
  };

  const adjustWallet = async () => {
    try {
      const values = await adjustForm.validateFields();
      await fetchData(BOUNCECAST_STARS_WALLET_ADJUST, { method: 'POST', data: values });
      message.success('Wallet adjusted');
      adjustForm.resetFields();
      loadStars();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to adjust wallet');
    }
  };

  const testOverlay = async () => {
    setTestingOverlay(true);
    try {
      const values = await overlayTestForm.validateFields();
      await fetchData(BOUNCECAST_STARS_TEST_OVERLAY, { method: 'POST', data: values });
      message.success('Stars overlay test sent');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to test overlay');
    } finally {
      setTestingOverlay(false);
    }
  };

  const packages = summary?.packages || [];

  return (
    <div className="bouncecast-studio-dashboard">
      <Title level={1}>Stars</Title>
      <Text type="secondary">
        Site-support gifting with PayPal checkout, chat events, and live overlays. Stars have no
        cash value and are not paid out to streamers.
      </Text>

      <Alert
        style={{ margin: '1rem 0' }}
        type="info"
        showIcon
        message="PayPal money goes to the site owner account configured here. No creator payout or withdrawal features are included."
      />

      <Tabs
        items={[
          {
            key: 'settings',
            label: 'Settings',
            children: (
              <>
                <Card title="PayPal webhook setup" style={{ marginBottom: 16 }} loading={loading}>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Text>
                      Add this webhook URL in your PayPal app, then paste the PayPal webhook ID into
                      the field below.
                    </Text>
                    <Typography.Paragraph copyable={{ text: paypalWebhookUrl }}>
                      <code>{paypalWebhookUrl}</code>
                    </Typography.Paragraph>
                    <Text type="secondary">
                      In PayPal Developer, create or open your app, copy the client ID and secret
                      into this page, create a webhook using the same sandbox/live environment, and
                      subscribe to PAYMENT.CAPTURE.COMPLETED, PAYMENT.CAPTURE.DENIED,
                      PAYMENT.CAPTURE.DECLINED, PAYMENT.CAPTURE.REFUNDED, and
                      PAYMENT.CAPTURE.REVERSED. Save the webhook, copy its webhook ID here, then
                      enable Stars and create at least one active package.
                    </Text>
                  </Space>
                </Card>
                <Card loading={loading}>
                  <Form form={settingsForm} layout="vertical" initialValues={defaultSettings}>
                    <Row gutter={16}>
                      <Col xs={24} md={8}>
                        <Form.Item name="enabled" label="Enable Stars" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={8}>
                        <Form.Item
                          name="paypalEnvironment"
                          label="PayPal environment"
                          rules={[{ required: true }]}
                        >
                          <Select
                            options={[
                              { label: 'Sandbox', value: 'sandbox' },
                              { label: 'Live', value: 'live' },
                            ]}
                          />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={8}>
                        <Form.Item name="currency" label="Currency" rules={[{ required: true }]}>
                          <Input maxLength={3} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={16}>
                      <Col xs={24} md={8}>
                        <Form.Item name="paypalClientId" label="PayPal client ID">
                          <Input />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={8}>
                        <Form.Item name="paypalClientSecret" label="PayPal client secret">
                          <Input.Password placeholder="Leave blank to keep existing secret" />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={8}>
                        <Form.Item name="paypalWebhookId" label="PayPal webhook ID">
                          <Input />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item name="supportMessage" label="Support message">
                      <Input.TextArea rows={3} maxLength={280} />
                    </Form.Item>
                    <Row gutter={16}>
                      <Col xs={24} md={6}>
                        <Form.Item name="minimumSendAmount" label="Minimum send">
                          <InputNumber min={1} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="maximumSendAmount" label="Maximum send">
                          <InputNumber min={1} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="sendCooldownSeconds" label="Send cooldown seconds">
                          <InputNumber min={0} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Space direction="vertical">
                          <Form.Item
                            name="overlayEffectsEnabled"
                            label="Overlay effects"
                            valuePropName="checked"
                          >
                            <Switch />
                          </Form.Item>
                          <Form.Item
                            name="soundEffectsEnabled"
                            label="Sound effects"
                            valuePropName="checked"
                          >
                            <Switch />
                          </Form.Item>
                        </Space>
                      </Col>
                    </Row>
                    <Button type="primary" onClick={saveSettings} loading={saving}>
                      Save Stars settings
                    </Button>
                  </Form>
                </Card>
                <Card title="Overlay test" style={{ marginTop: 16 }} loading={loading}>
                  <Form
                    form={overlayTestForm}
                    layout="vertical"
                    initialValues={{
                      displayName: 'BounceCast Test',
                      amount: 100,
                      message: 'Big up the stream',
                      effect: 'fireworks',
                      soundEnabled: summary?.settings?.soundEffectsEnabled ?? true,
                    }}
                  >
                    <Row gutter={16}>
                      <Col xs={24} md={6}>
                        <Form.Item name="displayName" label="Display name">
                          <Input maxLength={30} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="amount" label="Stars" rules={[{ required: true }]}>
                          <InputNumber min={1} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="effect" label="Effect" rules={[{ required: true }]}>
                          <Select options={starEffectOptions} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="soundEnabled" label="Sound" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item name="message" label="Message">
                      <Input maxLength={120} />
                    </Form.Item>
                    <Button onClick={testOverlay} loading={testingOverlay}>
                      Play overlay test
                    </Button>
                  </Form>
                </Card>
              </>
            ),
          },
          {
            key: 'packages',
            label: 'Packages',
            children: (
              <Card
                title="Star packages"
                extra={<Button onClick={() => openPackageModal()}>Add package</Button>}
              >
                <Table
                  rowKey="id"
                  dataSource={packages}
                  pagination={false}
                  columns={[
                    { title: 'Name', dataIndex: 'name' },
                    { title: 'Stars', dataIndex: 'starAmount' },
                    { title: 'Price', render: (_, record) => formatPrice(record) },
                    { title: 'Order', dataIndex: 'displayOrder' },
                    {
                      title: 'Status',
                      render: (_, record) => (
                        <Tag color={record.enabled ? 'green' : 'default'}>
                          {record.enabled ? 'Enabled' : 'Disabled'}
                        </Tag>
                      ),
                    },
                    {
                      title: '',
                      render: (_, record) => (
                        <Button size="small" onClick={() => openPackageModal(record)}>
                          Edit
                        </Button>
                      ),
                    },
                  ]}
                />
              </Card>
            ),
          },
          {
            key: 'logs',
            label: 'Logs',
            children: (
              <Row gutter={[16, 16]}>
                <Col xs={24} lg={12}>
                  <Card title="Wallets and manual adjustments" style={{ marginBottom: 16 }}>
                    <Form form={adjustForm} layout="vertical">
                      <Row gutter={12}>
                        <Col span={10}>
                          <Form.Item name="userId" label="User ID" rules={[{ required: true }]}>
                            <Input />
                          </Form.Item>
                        </Col>
                        <Col span={6}>
                          <Form.Item name="amount" label="Amount" rules={[{ required: true }]}>
                            <InputNumber style={{ width: '100%' }} />
                          </Form.Item>
                        </Col>
                        <Col span={8}>
                          <Form.Item name="notes" label="Notes">
                            <Input />
                          </Form.Item>
                        </Col>
                      </Row>
                      <Button onClick={adjustWallet}>Adjust wallet</Button>
                    </Form>
                    <Table
                      size="small"
                      rowKey="userId"
                      dataSource={summary?.wallets || []}
                      columns={[
                        { title: 'User ID', dataIndex: 'userId' },
                        { title: 'Balance', dataIndex: 'balance' },
                        { title: 'Purchased', dataIndex: 'lifetimePurchased' },
                        { title: 'Sent', dataIndex: 'lifetimeSent' },
                      ]}
                    />
                  </Card>
                  <Card title="PayPal orders">
                    <Table
                      size="small"
                      rowKey="id"
                      dataSource={summary?.orders || []}
                      columns={[
                        { title: 'Order ID', dataIndex: 'paypalOrderId' },
                        { title: 'Capture', dataIndex: 'paypalCaptureId' },
                        { title: 'Status', dataIndex: 'status' },
                        { title: 'Stars', dataIndex: 'starAmount' },
                      ]}
                    />
                  </Card>
                </Col>
                <Col xs={24} lg={12}>
                  <Card title="Leaderboard" style={{ marginBottom: 16 }}>
                    <Table
                      size="small"
                      rowKey="userId"
                      dataSource={summary?.leaderboard || []}
                      columns={[
                        {
                          title: 'Rank',
                          dataIndex: 'rank',
                          render: (rank: number) => <Tag color={rankTagColor(rank)}>{rank}</Tag>,
                        },
                        { title: 'User', dataIndex: 'displayName' },
                        { title: 'Total sent', dataIndex: 'totalSent' },
                        { title: 'Sends', dataIndex: 'sendCount' },
                      ]}
                    />
                  </Card>
                  <Card title="Star sends">
                    <Table
                      size="small"
                      rowKey="id"
                      dataSource={summary?.sendEvents || []}
                      columns={[
                        { title: 'User', dataIndex: 'displayName' },
                        { title: 'Stars', dataIndex: 'amount' },
                        { title: 'Effect', dataIndex: 'effect' },
                        { title: 'Message', dataIndex: 'message' },
                      ]}
                    />
                  </Card>
                </Col>
              </Row>
            ),
          },
        ]}
      />

      <Modal
        title={editingPackage ? 'Edit Star package' : 'Add Star package'}
        open={packageModalOpen}
        onOk={savePackage}
        onCancel={() => setPackageModalOpen(false)}
      >
        <Form form={packageForm} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="starAmount" label="Stars" rules={[{ required: true }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="priceCents" label="Price in cents" rules={[{ required: true }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="currency" label="Currency">
                <Input maxLength={3} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="displayOrder" label="Display order">
                <InputNumber style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="enabled" label="Enabled" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
}

StarsAdmin.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
