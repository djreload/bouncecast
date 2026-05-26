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
  BOUNCECAST_REWARDS_ACHIEVEMENTS,
  BOUNCECAST_REWARDS_ADMIN,
  BOUNCECAST_REWARDS_CREDITS_ADJUST,
  BOUNCECAST_REWARDS_MESSAGE_READ,
  BOUNCECAST_REWARDS_ORDER_DISPATCH,
  BOUNCECAST_REWARDS_ORDER_UPDATE,
  BOUNCECAST_REWARDS_ORDERS_EXPORT,
  BOUNCECAST_REWARDS_PRIZES,
  BOUNCECAST_REWARDS_SETTINGS,
  BOUNCECAST_REWARDS_TASKS,
  BOUNCECAST_REWARDS_TOP_SUPPORTERS_AWARD,
  fetchData,
} from '../../utils/apis';
import {
  RewardAchievement,
  RewardAdminSummary,
  RewardOrder,
  RewardPrize,
  RewardSettings,
  RewardTask,
  RewardTopSupporterAwardResult,
} from '../../interfaces/rewards.model';

const { Title, Text } = Typography;

const defaultSettings: RewardSettings = {
  enabled: false,
  spinCost: 1,
  chatRewardsEnabled: false,
  chatValidMessageCount: 10,
  chatCooldownSeconds: 300,
  chatCreditReward: 1,
  topSupporterFirstCredits: 25,
  topSupporterSecondCredits: 15,
  topSupporterThirdCredits: 10,
  overlayEnabled: true,
  overlayDurationSeconds: 5,
  overlaySoundEnabled: true,
  overlayShowImage: true,
  overlayTemplate: '{viewer} just won {prize} on the Rewards Wheel!',
};

function statusColor(status: string) {
  if (status === 'ready_to_fulfil') return 'green';
  if (status === 'awaiting_claim_details') return 'gold';
  if (status === 'dispatched' || status === 'delivered') return 'blue';
  if (status === 'cancelled') return 'red';
  return 'default';
}

function activePrizeCount(summary?: RewardAdminSummary) {
  return (summary?.prizes || []).filter(
    prize =>
      prize.active &&
      prize.oddsWeight > 0 &&
      (prize.prizeType === 'sorry' ||
        prize.stockQuantity === undefined ||
        prize.stockQuantity === null ||
        prize.stockQuantity > 0),
  ).length;
}

function activeTaskCount(summary?: RewardAdminSummary) {
  return (summary?.tasks || []).filter(task => task.active).length;
}

export default function RewardsAdmin() {
  const [summary, setSummary] = useState<RewardAdminSummary>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [prizeModalOpen, setPrizeModalOpen] = useState(false);
  const [editingPrize, setEditingPrize] = useState<RewardPrize>(null);
  const [orderModalOpen, setOrderModalOpen] = useState(false);
  const [editingOrder, setEditingOrder] = useState<RewardOrder>(null);
  const [taskModalOpen, setTaskModalOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<RewardTask>(null);
  const [achievementModalOpen, setAchievementModalOpen] = useState(false);
  const [editingAchievement, setEditingAchievement] = useState<RewardAchievement>(null);
  const [topSupporterResults, setTopSupporterResults] = useState<RewardTopSupporterAwardResult[]>(
    [],
  );
  const [settingsForm] = Form.useForm<RewardSettings>();
  const [prizeForm] = Form.useForm<RewardPrize>();
  const [creditForm] = Form.useForm();
  const [orderForm] = Form.useForm<RewardOrder>();
  const [dispatchForm] = Form.useForm();
  const [taskForm] = Form.useForm<RewardTask>();
  const [achievementForm] = Form.useForm<RewardAchievement>();

  const loadRewards = async () => {
    setLoading(true);
    try {
      const result = await fetchData(BOUNCECAST_REWARDS_ADMIN);
      setSummary(result);
      settingsForm.setFieldsValue({ ...defaultSettings, ...result.settings });
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load Rewards');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRewards();
  }, []);

  const saveSettings = async () => {
    setSaving(true);
    try {
      const values = await settingsForm.validateFields();
      await fetchData(BOUNCECAST_REWARDS_SETTINGS, { method: 'POST', data: values });
      message.success('Rewards settings saved');
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save settings');
    } finally {
      setSaving(false);
    }
  };

  const openPrize = (prize?: RewardPrize) => {
    setEditingPrize(prize || null);
    prizeForm.setFieldsValue(
      prize || {
        name: '',
        description: '',
        image: '',
        prizeType: 'sorry',
        oddsWeight: 100,
        active: true,
        displayOrder: 100,
        claimRequired: false,
        marketingConsentRequired: false,
      },
    );
    setPrizeModalOpen(true);
  };

  const savePrize = async () => {
    try {
      const values = await prizeForm.validateFields();
      await fetchData(BOUNCECAST_REWARDS_PRIZES, {
        method: 'POST',
        data: { ...values, id: editingPrize?.id || 0 },
      });
      message.success('Prize saved');
      setPrizeModalOpen(false);
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save prize');
    }
  };

  const adjustCredits = async () => {
    try {
      const values = await creditForm.validateFields();
      await fetchData(BOUNCECAST_REWARDS_CREDITS_ADJUST, { method: 'POST', data: values });
      message.success('Spin Credits updated');
      creditForm.resetFields();
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to adjust credits');
    }
  };

  const openOrder = (order: RewardOrder) => {
    setEditingOrder(order);
    orderForm.setFieldsValue(order);
    dispatchForm.setFieldsValue(order);
    setOrderModalOpen(true);
  };

  const saveOrder = async () => {
    try {
      const values = await orderForm.validateFields();
      await fetchData(BOUNCECAST_REWARDS_ORDER_UPDATE, {
        method: 'POST',
        data: { ...editingOrder, ...values },
      });
      message.success('Order updated');
      setOrderModalOpen(false);
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to update order');
    }
  };

  const dispatchOrder = async () => {
    try {
      const values = await dispatchForm.validateFields();
      await fetchData(BOUNCECAST_REWARDS_ORDER_DISPATCH, {
        method: 'POST',
        data: { ...values, orderId: editingOrder.id },
      });
      message.success('Order marked dispatched');
      setOrderModalOpen(false);
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to dispatch order');
    }
  };

  const openTask = (task?: RewardTask) => {
    setEditingTask(task || null);
    taskForm.setFieldsValue(task || { title: '', description: '', creditReward: 1, active: true });
    setTaskModalOpen(true);
  };

  const saveTask = async () => {
    const values = await taskForm.validateFields();
    await fetchData(BOUNCECAST_REWARDS_TASKS, {
      method: 'POST',
      data: { ...values, id: editingTask?.id || 0 },
    });
    setTaskModalOpen(false);
    setEditingTask(null);
    taskForm.resetFields();
    loadRewards();
  };

  const openAchievement = (achievement?: RewardAchievement) => {
    setEditingAchievement(achievement || null);
    achievementForm.setFieldsValue(
      achievement || { name: '', conditionKey: '', rewardAmount: 1, active: true },
    );
    setAchievementModalOpen(true);
  };

  const saveAchievement = async () => {
    const values = await achievementForm.validateFields();
    await fetchData(BOUNCECAST_REWARDS_ACHIEVEMENTS, {
      method: 'POST',
      data: { ...values, id: editingAchievement?.id || 0 },
    });
    setAchievementModalOpen(false);
    setEditingAchievement(null);
    achievementForm.resetFields();
    loadRewards();
  };

  const markMessageRead = async (id: number) => {
    await fetchData(BOUNCECAST_REWARDS_MESSAGE_READ, { method: 'POST', data: { id } });
    loadRewards();
  };

  const awardTopSupporters = async () => {
    try {
      const response = await fetchData(BOUNCECAST_REWARDS_TOP_SUPPORTERS_AWARD, {
        method: 'POST',
        data: {},
      });
      const results = response.results || [];
      setTopSupporterResults(results);
      const awardedCount = results.filter(result => result.awarded).length;
      message.success(
        awardedCount
          ? `Awarded ${awardedCount} top supporter reward${awardedCount === 1 ? '' : 's'}`
          : 'No new top supporter rewards were awarded',
      );
      loadRewards();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to award top supporters');
    }
  };

  const activePrizes = activePrizeCount(summary);
  const activeTasks = activeTaskCount(summary);
  const hasCredits = (summary?.balances || []).some(balance => balance.balance > 0);
  const hasCreditPath = Boolean(
    summary?.settings?.chatRewardsEnabled || activeTasks > 0 || hasCredits,
  );
  const wheelEnabled = Boolean(summary?.settings?.enabled);

  return (
    <div className="bouncecast-studio-dashboard">
      <Title level={1}>Rewards Wheel</Title>
      <Text type="secondary">
        Internal Spin Credits, weighted wheel prizes, claim details, and manual fulfilment. No
        payments, checkout, shops, or streamer payouts are included.
      </Text>
      <Alert
        style={{ margin: '1rem 0' }}
        type="info"
        showIcon
        message="Real prizes create internal fulfilment orders only. Discount and paid reward paths are placeholders for future approval."
      />
      <Card loading={loading} style={{ margin: '1rem 0' }}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Space wrap>
            <Tag color={wheelEnabled ? 'green' : 'gold'}>
              Wheel {wheelEnabled ? 'enabled' : 'disabled'}
            </Tag>
            <Tag color={activePrizes > 0 ? 'green' : 'red'}>
              {activePrizes} active prize{activePrizes === 1 ? '' : 's'}
            </Tag>
            <Tag color={hasCreditPath ? 'green' : 'gold'}>
              {hasCreditPath ? 'Spin Credits available' : 'No credit path yet'}
            </Tag>
          </Space>
          <Text>
            To activate the Rewards Wheel for viewers, turn on the wheel, add at least one active
            prize with odds above zero, then give viewers Spin Credits through chat rewards, tasks,
            top-supporter awards, or the Spin Credits tab. For the mobile app, also enable Rewards
            Wheel in the Mobile App settings so the in-app panel appears.
          </Text>
          <Text type="secondary">
            Viewers use <Text code>/rewards</Text> on the website, and mobile users tap Rewards to
            see prizes, tasks, balance, and the Spin button.
          </Text>
        </Space>
      </Card>

      <Tabs
        items={[
          {
            key: 'settings',
            label: 'Settings',
            children: (
              <Card loading={loading}>
                <Form form={settingsForm} layout="vertical" initialValues={defaultSettings}>
                  <Row gutter={16}>
                    <Col xs={24} md={6}>
                      <Form.Item name="enabled" label="Enable wheel" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name="spinCost" label="Credits per spin">
                        <InputNumber min={1} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name="overlayEnabled" label="Win overlay" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name="overlaySoundEnabled"
                        label="Overlay sound"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Form.Item name="overlayTemplate" label="Overlay template">
                    <Input maxLength={180} />
                  </Form.Item>
                  <Row gutter={16}>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name="chatRewardsEnabled"
                        label="Chat rewards"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name="chatValidMessageCount" label="Messages needed">
                        <InputNumber min={1} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name="chatCooldownSeconds" label="Chat cooldown">
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name="chatCreditReward" label="Chat credit reward">
                        <InputNumber min={1} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Row gutter={16}>
                    <Col xs={24} md={8}>
                      <Form.Item name="topSupporterFirstCredits" label="Top supporter 1st">
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="topSupporterSecondCredits" label="Top supporter 2nd">
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="topSupporterThirdCredits" label="Top supporter 3rd">
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Button type="primary" onClick={saveSettings} loading={saving}>
                    Save settings
                  </Button>
                </Form>
              </Card>
            ),
          },
          {
            key: 'prizes',
            label: 'Prizes',
            children: (
              <>
                <Button type="primary" onClick={() => openPrize()} style={{ marginBottom: 12 }}>
                  Add prize
                </Button>
                <Table
                  loading={loading}
                  rowKey="id"
                  dataSource={summary?.prizes || []}
                  columns={[
                    { title: 'Name', dataIndex: 'name' },
                    { title: 'Type', dataIndex: 'prizeType', render: value => <Tag>{value}</Tag> },
                    { title: 'Odds', dataIndex: 'oddsWeight' },
                    {
                      title: 'Stock',
                      render: (_, record: RewardPrize) =>
                        record.prizeType === 'sorry'
                          ? 'n/a'
                          : (record.stockQuantity ?? 'Unlimited'),
                    },
                    {
                      title: 'Active',
                      dataIndex: 'active',
                      render: value => (value ? 'Yes' : 'No'),
                    },
                    {
                      title: 'Actions',
                      render: (_, record: RewardPrize) => (
                        <Button size="small" onClick={() => openPrize(record)}>
                          Edit
                        </Button>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'credits',
            label: 'Spin Credits',
            children: (
              <>
                <Card style={{ marginBottom: 16 }}>
                  <Form form={creditForm} layout="inline">
                    <Form.Item name="userId" label="User ID" rules={[{ required: true }]}>
                      <Input style={{ width: 260 }} />
                    </Form.Item>
                    <Form.Item name="amount" label="Amount" rules={[{ required: true }]}>
                      <InputNumber />
                    </Form.Item>
                    <Form.Item name="note" label="Note">
                      <Input style={{ width: 280 }} />
                    </Form.Item>
                    <Button type="primary" onClick={adjustCredits}>
                      Adjust
                    </Button>
                  </Form>
                </Card>
                <Table
                  loading={loading}
                  rowKey="userId"
                  dataSource={summary?.balances || []}
                  columns={[
                    { title: 'User', dataIndex: 'displayName' },
                    { title: 'User ID', dataIndex: 'userId' },
                    { title: 'Balance', dataIndex: 'balance' },
                    { title: 'Earned', dataIndex: 'lifetimeEarned' },
                    { title: 'Spent', dataIndex: 'lifetimeSpent' },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'orders',
            label: 'Fulfilment',
            children: (
              <>
                <Button href={BOUNCECAST_REWARDS_ORDERS_EXPORT} style={{ marginBottom: 12 }}>
                  Export CSV
                </Button>
                <Table
                  loading={loading}
                  rowKey="id"
                  dataSource={summary?.orders || []}
                  columns={[
                    { title: 'Order', dataIndex: 'id' },
                    { title: 'Viewer', dataIndex: 'usernameSnapshot' },
                    { title: 'Prize', dataIndex: 'prizeSnapshot' },
                    {
                      title: 'Status',
                      dataIndex: 'orderStatus',
                      render: value => <Tag color={statusColor(value)}>{value}</Tag>,
                    },
                    { title: 'Courier', dataIndex: 'courier' },
                    {
                      title: 'Actions',
                      render: (_, record: RewardOrder) => (
                        <Button size="small" onClick={() => openOrder(record)}>
                          Manage
                        </Button>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'messages',
            label: `Messages${summary?.unreadCount ? ` (${summary.unreadCount})` : ''}`,
            children: (
              <Table
                loading={loading}
                rowKey="id"
                dataSource={summary?.adminMessages || []}
                columns={[
                  { title: 'Title', dataIndex: 'title' },
                  { title: 'Body', dataIndex: 'body' },
                  { title: 'Order', dataIndex: 'orderId' },
                  {
                    title: 'State',
                    render: (_, record) =>
                      record.readAt ? <Tag>Read</Tag> : <Tag color="gold">Unread</Tag>,
                  },
                  {
                    title: 'Actions',
                    render: (_, record) =>
                      !record.readAt && (
                        <Button size="small" onClick={() => markMessageRead(record.id)}>
                          Mark read
                        </Button>
                      ),
                  },
                ]}
              />
            ),
          },
          {
            key: 'automation',
            label: 'Tasks & Achievements',
            children: (
              <Row gutter={16}>
                <Col xs={24}>
                  <Card style={{ marginBottom: 16 }}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Space wrap align="center">
                        <Button type="primary" onClick={awardTopSupporters}>
                          Award top Stars supporters
                        </Button>
                        <Text type="secondary">
                          Grants today&apos;s configured Spin Credit rewards to the current Stars
                          leaderboard top 3. Each rank can only be awarded once per day.
                        </Text>
                      </Space>
                      {topSupporterResults.length > 0 && (
                        <Table
                          size="small"
                          rowKey={record => `${record.rank}-${record.userId || record.reason}`}
                          dataSource={topSupporterResults}
                          pagination={false}
                          columns={[
                            { title: 'Rank', dataIndex: 'rank' },
                            { title: 'Viewer', dataIndex: 'displayName' },
                            { title: 'Stars sent', dataIndex: 'totalSent' },
                            { title: 'Credits', dataIndex: 'credits' },
                            {
                              title: 'Result',
                              render: (_, record: RewardTopSupporterAwardResult) =>
                                record.awarded ? (
                                  <Tag color="green">Awarded</Tag>
                                ) : (
                                  <Tag color="gold">{record.reason || 'Skipped'}</Tag>
                                ),
                            },
                          ]}
                        />
                      )}
                    </Space>
                  </Card>
                </Col>
                <Col xs={24} md={12}>
                  <Button onClick={() => openTask()} style={{ marginBottom: 12 }}>
                    Add task
                  </Button>
                  <Table
                    rowKey="id"
                    dataSource={summary?.tasks || []}
                    columns={[
                      { title: 'Task', dataIndex: 'title' },
                      { title: 'Credits', dataIndex: 'creditReward' },
                      {
                        title: 'Active',
                        dataIndex: 'active',
                        render: value => (value ? 'Yes' : 'No'),
                      },
                      {
                        title: 'Actions',
                        render: (_, record: RewardTask) => (
                          <Button size="small" onClick={() => openTask(record)}>
                            Edit
                          </Button>
                        ),
                      },
                    ]}
                  />
                </Col>
                <Col xs={24} md={12}>
                  <Button onClick={() => openAchievement()} style={{ marginBottom: 12 }}>
                    Add achievement
                  </Button>
                  <Table
                    rowKey="id"
                    dataSource={summary?.achievements || []}
                    columns={[
                      { title: 'Achievement', dataIndex: 'name' },
                      { title: 'Condition', dataIndex: 'conditionKey' },
                      { title: 'Credits', dataIndex: 'rewardAmount' },
                      {
                        title: 'Actions',
                        render: (_, record: RewardAchievement) => (
                          <Button size="small" onClick={() => openAchievement(record)}>
                            Edit
                          </Button>
                        ),
                      },
                    ]}
                  />
                </Col>
                <Col xs={24} md={12} style={{ marginTop: 16 }}>
                  <Title level={4}>Recent task completions</Title>
                  <Table
                    rowKey="id"
                    dataSource={summary?.taskCompletions || []}
                    columns={[
                      { title: 'User ID', dataIndex: 'userId' },
                      { title: 'Task', dataIndex: 'taskTitle' },
                      { title: 'Status', dataIndex: 'status' },
                      { title: 'Completed', dataIndex: 'completedAt' },
                    ]}
                  />
                </Col>
                <Col xs={24} md={12} style={{ marginTop: 16 }}>
                  <Title level={4}>Recent achievement unlocks</Title>
                  <Table
                    rowKey="id"
                    dataSource={summary?.achievementUnlocks || []}
                    columns={[
                      { title: 'User ID', dataIndex: 'userId' },
                      { title: 'Achievement', dataIndex: 'achievementName' },
                      { title: 'Condition', dataIndex: 'conditionKey' },
                      { title: 'Unlocked', dataIndex: 'unlockedAt' },
                    ]}
                  />
                </Col>
              </Row>
            ),
          },
        ]}
      />

      <Modal
        open={prizeModalOpen}
        title="Reward prize"
        onCancel={() => setPrizeModalOpen(false)}
        onOk={savePrize}
      >
        <Form form={prizeForm} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="image" label="Image URL/path">
            <Input />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="prizeType" label="Type">
                <Select
                  options={[
                    { label: 'Physical', value: 'physical' },
                    { label: 'Digital', value: 'digital' },
                    { label: 'Discount placeholder', value: 'discount_future_placeholder' },
                    { label: 'Sorry', value: 'sorry' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="oddsWeight" label="Odds weight">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="stockQuantity" label="Stock quantity">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="displayOrder" label="Display order">
                <InputNumber style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Space wrap>
            <Form.Item name="active" label="Active" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="claimRequired" label="Claim required" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item
              name="marketingConsentRequired"
              label="Marketing consent required"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Space>
          <Form.Item name="terms" label="Terms">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="fulfilmentNotes" label="Private fulfilment notes">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={orderModalOpen}
        title={`Order #${editingOrder?.id || ''}`}
        onCancel={() => setOrderModalOpen(false)}
        footer={[
          <Button key="save" onClick={saveOrder}>
            Save
          </Button>,
          <Button key="dispatch" type="primary" onClick={dispatchOrder}>
            Mark dispatched
          </Button>,
        ]}
      >
        <Form form={orderForm} layout="vertical">
          <Form.Item name="orderStatus" label="Order status">
            <Select
              options={[
                'awaiting_claim_details',
                'ready_to_fulfil',
                'packed',
                'dispatched',
                'delivered',
                'cancelled',
              ].map(value => ({ label: value, value }))}
            />
          </Form.Item>
          <Form.Item name="adminNotes" label="Admin notes">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
        <Form form={dispatchForm} layout="vertical">
          <Form.Item name="courier" label="Courier">
            <Input />
          </Form.Item>
          <Form.Item name="trackingReference" label="Tracking reference">
            <Input />
          </Form.Item>
          <Form.Item name="trackingUrl" label="Tracking URL">
            <Input />
          </Form.Item>
          <Form.Item name="dispatchNote" label="Dispatch note to winner">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={taskModalOpen}
        title={editingTask ? 'Edit reward task' : 'Reward task'}
        onCancel={() => {
          setTaskModalOpen(false);
          setEditingTask(null);
        }}
        onOk={saveTask}
      >
        <Form form={taskForm} layout="vertical" initialValues={{ creditReward: 1, active: true }}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="creditReward" label="Credit reward">
            <InputNumber min={1} />
          </Form.Item>
          <Form.Item name="active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={achievementModalOpen}
        title={editingAchievement ? 'Edit reward achievement' : 'Reward achievement'}
        onCancel={() => {
          setAchievementModalOpen(false);
          setEditingAchievement(null);
        }}
        onOk={saveAchievement}
      >
        <Form
          form={achievementForm}
          layout="vertical"
          initialValues={{ rewardAmount: 1, active: true }}
        >
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="conditionKey" label="Condition key" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'First Rewards Wheel spin', value: 'reward_first_spin' },
                { label: 'Rewards Wheel prize win', value: 'reward_prize_win' },
                { label: 'Reward task completed', value: 'reward_task_completed' },
                { label: 'Stars sent', value: 'stars_sent' },
                { label: 'Top supporter reward granted', value: 'top_supporter_reward' },
              ]}
            />
          </Form.Item>
          <Form.Item name="rewardAmount" label="Reward amount">
            <InputNumber min={1} />
          </Form.Item>
          <Form.Item name="active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

RewardsAdmin.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
