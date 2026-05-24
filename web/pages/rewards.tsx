import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  Alert,
  Button,
  Checkbox,
  Form,
  Input,
  Modal,
  Space,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import { ACCESS_TOKEN_KEY } from '../components/stores/ClientConfigStore';
import {
  REWARDS_CLAIM_SUBMIT,
  REWARDS_CLAIMS,
  REWARDS_HISTORY,
  REWARDS_NOTIFICATION_READ,
  REWARDS_NOTIFICATIONS,
  REWARDS_SPIN,
  REWARDS_TASK_COMPLETE,
  REWARDS_WHEEL,
  getUnauthedData,
} from '../utils/apis';
import {
  RewardClaim,
  RewardNotification,
  RewardSpin,
  RewardSpinResult,
  RewardWheelData,
} from '../interfaces/rewards.model';
import styles from '../styles/rewards.module.scss';

const { Title, Text } = Typography;

function withToken(url: string, token: string) {
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}accessToken=${encodeURIComponent(token)}`;
}

function formatDate(value?: string) {
  if (!value) return '';
  return new Date(value).toLocaleString([], {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function RewardsPage() {
  const [token, setToken] = useState('');
  const [wheelData, setWheelData] = useState<RewardWheelData>(null);
  const [history, setHistory] = useState<RewardSpin[]>([]);
  const [claims, setClaims] = useState<RewardClaim[]>([]);
  const [notifications, setNotifications] = useState<RewardNotification[]>([]);
  const [loading, setLoading] = useState(true);
  const [spinning, setSpinning] = useState(false);
  const [rotation, setRotation] = useState(0);
  const [result, setResult] = useState<RewardSpinResult>(null);
  const [claimModalOpen, setClaimModalOpen] = useState(false);
  const [activeClaim, setActiveClaim] = useState<RewardClaim>(null);
  const [claimForm] = Form.useForm();

  const loadRewards = async (activeToken: string) => {
    if (!activeToken) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [wheel, spins, myClaims, myNotifications] = await Promise.all([
        getUnauthedData(withToken(REWARDS_WHEEL, activeToken)),
        getUnauthedData(withToken(REWARDS_HISTORY, activeToken)),
        getUnauthedData(withToken(REWARDS_CLAIMS, activeToken)),
        getUnauthedData(withToken(REWARDS_NOTIFICATIONS, activeToken)),
      ]);
      setWheelData(wheel);
      setHistory(spins || []);
      setClaims(myClaims || []);
      setNotifications(myNotifications || []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load rewards');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const savedToken = localStorage.getItem(ACCESS_TOKEN_KEY) || '';
    setToken(savedToken);
    loadRewards(savedToken);
  }, []);

  const eligiblePrizes = wheelData?.prizes || [];
  const rewardTasks = wheelData?.tasks || [];
  const completedTaskIDs = useMemo(
    () => new Set((wheelData?.taskCompletions || []).map(completion => completion.taskId)),
    [wheelData?.taskCompletions],
  );
  const highlightedPrize = result?.prize || eligiblePrizes[0];
  const enabled = Boolean(wheelData?.settings?.enabled);

  const wheelStyle = useMemo(
    () => ({
      transform: `rotate(${rotation}deg)`,
    }),
    [rotation],
  );

  const spin = async () => {
    setSpinning(true);
    setResult(null);
    setRotation(current => current + 720 + Math.floor(Math.random() * 360));
    try {
      const spinResult = await getUnauthedData(withToken(REWARDS_SPIN, token), {
        method: 'POST',
        data: {},
      });
      setTimeout(() => {
        setResult(spinResult);
        setWheelData(current =>
          current
            ? {
                ...current,
                balance: { ...current.balance, balance: spinResult.balance },
              }
            : current,
        );
        if (spinResult.claim && spinResult.claim.status === 'pending_details') {
          setActiveClaim(spinResult.claim);
          claimForm.resetFields();
          setClaimModalOpen(true);
        }
        loadRewards(token);
      }, 1100);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to spin');
    } finally {
      setSpinning(false);
    }
  };

  const openClaim = (claim: RewardClaim) => {
    setActiveClaim(claim);
    claimForm.setFieldsValue(claim);
    setClaimModalOpen(true);
  };

  const submitClaim = async () => {
    const values = await claimForm.validateFields();
    try {
      await getUnauthedData(withToken(REWARDS_CLAIM_SUBMIT, token), {
        method: 'POST',
        data: { ...values, claimId: activeClaim.id, consentText: activeClaim.consentText },
      });
      message.success('Claim details saved');
      setClaimModalOpen(false);
      loadRewards(token);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save claim');
    }
  };

  const markNotificationRead = async (notification: RewardNotification) => {
    await getUnauthedData(withToken(REWARDS_NOTIFICATION_READ, token), {
      method: 'POST',
      data: { id: notification.id },
    });
    loadRewards(token);
  };

  const completeTask = async (taskId: number) => {
    try {
      const response = await getUnauthedData(withToken(REWARDS_TASK_COMPLETE, token), {
        method: 'POST',
        data: { taskId },
      });
      if (response.awarded) {
        message.success('Spin Credits added');
      } else {
        message.info('Task already completed');
      }
      loadRewards(token);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to complete task');
    }
  };

  if (!token) {
    return (
      <div className={styles.page}>
        <div className={styles.shell}>
          <Title className={styles.title}>Rewards Wheel</Title>
          <Alert
            type="info"
            showIcon
            message="Log in to earn Spin Credits and claim prizes."
            action={
              <Link href="/login?next=/rewards">
                <Button type="primary">Log in / register</Button>
              </Link>
            }
          />
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <div className={styles.shell}>
        <div className={styles.hero}>
          <div>
            <h1 className={styles.title}>Rewards Wheel</h1>
            <p className={styles.subtitle}>
              Earn Spin Credits from BounceCast activity and spend them on the internal prize wheel.
              There is no checkout here: prizes are fulfilled manually by the site team.
            </p>
          </div>
          <div className={styles.balance}>
            <span>Spin Credits</span>
            <strong>{wheelData?.balance?.balance || 0}</strong>
          </div>
        </div>

        {!enabled && (
          <Alert
            style={{ marginBottom: '1rem' }}
            type="warning"
            showIcon
            message="The Rewards Wheel is currently disabled by the site team."
          />
        )}

        <div className={styles.grid}>
          <section className={`${styles.panel} ${styles.wheelPanel}`}>
            <div className={styles.pointer} />
            <div className={styles.wheel} style={wheelStyle}>
              <div className={styles.wheelLabel}>
                <span>{spinning ? 'Spinning...' : 'Featured prize'}</span>
                <strong>{highlightedPrize?.name || 'No prizes yet'}</strong>
                <Text style={{ color: 'rgba(255,255,255,.72)' }}>
                  {highlightedPrize?.prizeType || 'waiting'}
                </Text>
              </div>
            </div>
            <Button
              className={styles.spinButton}
              type="primary"
              disabled={!enabled || spinning || !wheelData || wheelData.balance.balance < 1}
              loading={spinning || loading}
              onClick={spin}
            >
              Spin the wheel
            </Button>
            {result && (
              <Alert
                style={{ marginTop: '1rem', width: '100%' }}
                type={result.winner ? 'success' : 'info'}
                showIcon
                message={result.message}
              />
            )}
          </section>

          <section className={styles.panel}>
            <Tabs
              items={[
                {
                  key: 'prizes',
                  label: 'Prizes',
                  children: (
                    <div className={styles.list}>
                      {eligiblePrizes.map(prize => (
                        <article className={styles.listItem} key={prize.id}>
                          <Space wrap>
                            <strong>{prize.name}</strong>
                            <Tag>{prize.prizeType}</Tag>
                            {prize.stockQuantity !== undefined && (
                              <Tag color={prize.stockQuantity > 0 ? 'green' : 'red'}>
                                Stock {prize.stockQuantity}
                              </Tag>
                            )}
                          </Space>
                          <p>{prize.description || prize.terms || 'Rewards Wheel prize'}</p>
                        </article>
                      ))}
                    </div>
                  ),
                },
                {
                  key: 'tasks',
                  label: 'Tasks',
                  children: (
                    <div className={styles.list}>
                      {rewardTasks.length === 0 && (
                        <article className={styles.listItem}>
                          <strong>No reward tasks are active yet.</strong>
                          <p>Check back when the site team adds new ways to earn Spin Credits.</p>
                        </article>
                      )}
                      {rewardTasks.map(task => {
                        const completed = completedTaskIDs.has(task.id);
                        return (
                          <article className={styles.listItem} key={task.id}>
                            <Space wrap>
                              <strong>{task.title}</strong>
                              <Tag color="cyan">+{task.creditReward} credits</Tag>
                              {completed ? (
                                <Tag color="green">Completed</Tag>
                              ) : (
                                <Button size="small" onClick={() => completeTask(task.id)}>
                                  Complete
                                </Button>
                              )}
                            </Space>
                            <p>{task.description || 'Complete this task to earn Spin Credits.'}</p>
                          </article>
                        );
                      })}
                    </div>
                  ),
                },
                {
                  key: 'history',
                  label: 'History',
                  children: (
                    <div className={styles.list}>
                      {history.map(spinItem => (
                        <article className={styles.listItem} key={spinItem.id}>
                          <Space wrap>
                            <strong>{spinItem.prizeSnapshot}</strong>
                            <Tag>{spinItem.resultType}</Tag>
                          </Space>
                          <p>{formatDate(spinItem.createdAt)}</p>
                        </article>
                      ))}
                    </div>
                  ),
                },
                {
                  key: 'claims',
                  label: 'Claims',
                  children: (
                    <div className={styles.list}>
                      {claims.map(claim => (
                        <article className={styles.listItem} key={claim.id}>
                          <Space wrap>
                            <strong>{claim.prizeSnapshot}</strong>
                            <Tag>{claim.status}</Tag>
                            {claim.status === 'pending_details' && (
                              <Button size="small" onClick={() => openClaim(claim)}>
                                Submit details
                              </Button>
                            )}
                          </Space>
                          <p>{formatDate(claim.createdAt)}</p>
                        </article>
                      ))}
                    </div>
                  ),
                },
                {
                  key: 'notifications',
                  label: 'Notifications',
                  children: (
                    <div className={styles.list}>
                      {notifications.map(notification => (
                        <article className={styles.listItem} key={notification.id}>
                          <Space wrap>
                            <strong>{notification.title}</strong>
                            {!notification.readAt && (
                              <Button
                                size="small"
                                onClick={() => markNotificationRead(notification)}
                              >
                                Mark read
                              </Button>
                            )}
                          </Space>
                          <p>{notification.message}</p>
                        </article>
                      ))}
                    </div>
                  ),
                },
              ]}
            />
          </section>
        </div>
      </div>

      <Modal
        open={claimModalOpen}
        title={`Claim ${activeClaim?.prizeSnapshot || 'your prize'}`}
        onCancel={() => setClaimModalOpen(false)}
        onOk={submitClaim}
        okText="Save claim details"
      >
        <Form form={claimForm} layout="vertical">
          <Form.Item name="fullName" label="Full name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="addressLine1" label="Address line 1" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="addressLine2" label="Address line 2">
            <Input />
          </Form.Item>
          <Space style={{ width: '100%' }} direction="vertical">
            <Form.Item name="townCity" label="Town / city" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name="countyState" label="County / state">
              <Input />
            </Form.Item>
            <Form.Item name="postcode" label="Postcode / ZIP" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name="country" label="Country" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          </Space>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label="Phone">
            <Input />
          </Form.Item>
          <Form.Item name="deliveryNotes" label="Delivery notes">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="marketingConsent" valuePropName="checked">
            <Checkbox>I agree to optional marketing contact for this prize.</Checkbox>
          </Form.Item>
          <Text type="secondary">{activeClaim?.consentText}</Text>
        </Form>
      </Modal>
    </div>
  );
}

RewardsPage.getLayout = function getLayout(page: ReactElement) {
  return page;
};
