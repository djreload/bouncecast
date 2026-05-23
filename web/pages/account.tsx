import React, { ReactElement, useEffect, useState } from 'react';
import Link from 'next/link';
import {
  Alert,
  Avatar,
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  Space,
  Statistic,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd';
import { AccountPayload, AccountService } from '../services/account-service';
import { registerWebPushNotifications } from '../services/notifications-service';
import { BOUNCECAST_AUTH_LOGOUT, getUnauthedData } from '../utils/apis';
import {
  BounceCastAccountHub,
  BounceCastDestination,
  BounceCastScheduleItem,
  BounceCastService,
} from '../services/bouncecast-service';
import { ACCESS_TOKEN_KEY } from '../components/stores/ClientConfigStore';
import styles from '../styles/bouncecast-public.module.scss';

const { Text } = Typography;

async function getBrowserPushEndpoint(enabled: boolean) {
  if (!enabled) {
    return '';
  }
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    throw new Error('Browser push notifications are not supported by this browser');
  }
  const response = await fetch('/api/config');
  const config = await response.json();
  const publicKey = config?.notifications?.browser?.publicKey;
  if (!config?.notifications?.browser?.enabled || !publicKey) {
    throw new Error('Browser push notifications are not enabled on this server yet');
  }
  return registerWebPushNotifications(publicKey);
}

function formatDate(value?: string) {
  if (!value) {
    return 'Not scheduled';
  }
  return new Date(value).toLocaleString([], {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function roleColor(role: string) {
  if (role === 'owner') return 'gold';
  if (role === 'admin') return 'magenta';
  if (role === 'moderator') return 'cyan';
  if (role === 'dj') return 'blue';
  return 'default';
}

function reminderStatusColor(reminder: { disabledAt?: string; lastDeliveryStatus?: string }) {
  if (reminder.disabledAt) {
    return 'default';
  }
  if (reminder.lastDeliveryStatus === 'sent') {
    return 'green';
  }
  if (reminder.lastDeliveryStatus === 'failed') {
    return 'red';
  }
  return 'blue';
}

function ScheduleList({
  schedule,
  accessToken,
  reminderChannels,
}: {
  schedule: BounceCastScheduleItem[];
  accessToken?: string;
  reminderChannels?: { email: boolean; browserPush: boolean; messenger: boolean };
}) {
  if (!schedule.length) {
    return <Text className={styles.muted}>No public sets are scheduled yet.</Text>;
  }

  const saveReminder = async (item: BounceCastScheduleItem) => {
    if (!accessToken || !reminderChannels) {
      return;
    }
    try {
      const browserPushEndpoint = await getBrowserPushEndpoint(
        Boolean(reminderChannels.browserPush),
      );
      await BounceCastService.setScheduleReminder(accessToken, item.id, {
        ...reminderChannels,
        browserPushEndpoint,
      });
      message.success(`Reminder saved for ${item.title}`);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save reminder');
    }
  };

  return (
    <div className={styles.scheduleList}>
      {schedule.map(item => (
        <article className={styles.scheduleItem} key={item.id}>
          <p className={styles.scheduleTime}>{formatDate(item.startsAt)}</p>
          <p className={styles.scheduleTitle}>{item.title}</p>
          <Text className={styles.muted}>
            {item.streamer ? `with ${item.streamer}` : 'BounceCast set'}
          </Text>
          {accessToken && reminderChannels && (
            <Button
              size="small"
              className={styles.reminderButton}
              onClick={() => saveReminder(item)}
            >
              Remind me
            </Button>
          )}
        </article>
      ))}
    </div>
  );
}

function DestinationButton({ destination }: { destination: BounceCastDestination }) {
  const button = (
    <Button type={destination.available ? 'primary' : 'default'} disabled={!destination.available}>
      {destination.label}
    </Button>
  );

  if (!destination.available) {
    return button;
  }

  return <Link href={destination.url}>{button}</Link>;
}

export default function AccountPage() {
  const [token, setToken] = useState('');
  const [hub, setHub] = useState<BounceCastAccountHub | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [profileForm] = Form.useForm<AccountPayload>();
  const [notificationForm] = Form.useForm();

  const loadHub = async (activeToken: string) => {
    if (!activeToken) {
      setHub(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const result = await BounceCastService.getAccountHub(activeToken);
      setHub(result);
      profileForm.setFieldsValue({
        displayName: result.user.displayName,
        profileImageUrl: result.user.profileImageUrl,
      });
      notificationForm.setFieldsValue(result.notificationPreferences);
    } catch {
      setHub(null);
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      setToken('');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const savedToken = localStorage.getItem(ACCESS_TOKEN_KEY) || '';
    setToken(savedToken);
    loadHub(savedToken);
  }, []);

  const applyAccountResponse = async result => {
    const nextToken = result?.accessToken || token;
    if (result?.accessToken) {
      localStorage.setItem(ACCESS_TOKEN_KEY, result.accessToken);
      setToken(result.accessToken);
    }
    await loadHub(nextToken);
  };

  const saveProfile = async () => {
    const values = await profileForm.validateFields();
    setSaving(true);
    try {
      const result = await AccountService.updateProfile(token, values);
      await applyAccountResponse(result);
      message.success('Profile saved');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save profile');
    } finally {
      setSaving(false);
    }
  };

  const saveNotifications = async () => {
    const values = await notificationForm.validateFields();
    setSaving(true);
    try {
      const result = await AccountService.updateNotifications(token, {
        ...values,
        browserPushEndpoint: await getBrowserPushEndpoint(Boolean(values.browserPush)),
      });
      await applyAccountResponse(result);
      message.success('Notifications saved');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save notifications');
    } finally {
      setSaving(false);
    }
  };

  const uploadProfileImage = async (file: File) => {
    if (!token) {
      message.error('Log in before uploading a profile picture');
      return;
    }
    setUploading(true);
    try {
      const result = await AccountService.uploadProfileImage(token, file);
      await applyAccountResponse(result);
      profileForm.setFieldsValue({ profileImageUrl: result?.user?.profileImageUrl });
      message.success('Profile picture uploaded');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to upload picture');
    } finally {
      setUploading(false);
    }
  };

  const logout = async () => {
    try {
      await getUnauthedData(BOUNCECAST_AUTH_LOGOUT, {
        method: 'POST',
        data: {
          accessToken: token,
          studioToken: localStorage.getItem('bouncecastStudioToken') || '',
        },
      });
    } catch (error) {
      console.error(error);
    }
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem('bouncecastStudioToken');
    setToken('');
    setHub(null);
  };

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <header className={styles.topbar}>
          <Link href="/" className={styles.brand}>
            <img src="/logo" alt="BounceCast" />
            <p className={styles.brandText}>BounceCast</p>
          </Link>
          <nav className={styles.nav} aria-label="Public navigation">
            <Link href="/">Live stream</Link>
            <Link href="/djs">DJ profiles</Link>
            <Link href="/login">Login</Link>
          </nav>
        </header>

        <section className={styles.hero}>
          <div>
            <p className={styles.eyebrow}>Unified Account Hub</p>
            <h1 className={styles.headline}>Your BounceCast account</h1>
            <p className={styles.copy}>
              Manage your chat profile, notification opt-ins, Stars wallet, and role-based dashboard
              access from one place.
            </p>
          </div>
          {hub && <Button onClick={logout}>Log out</Button>}
        </section>

        {!hub && !loading && (
          <Card className={styles.panel}>
            <Alert
              type="info"
              showIcon
              message="Use the unified BounceCast login"
              description="One login now covers your chat profile, notifications, Stars wallet, DJ Studio, and admin access when your role allows it."
              action={
                <Link href="/login?next=/account">
                  <Button type="primary">Open login</Button>
                </Link>
              }
            />
          </Card>
        )}

        {hub && (
          <div className={styles.twoColumn}>
            <section>
              <Card className={styles.panel}>
                <div className={styles.accountHeader}>
                  <Avatar src={hub.user.profileImageUrl} size={72}>
                    {hub.user.displayName?.slice(0, 1)}
                  </Avatar>
                  <div>
                    <p className={styles.accountName}>{hub.user.displayName}</p>
                    <Text className={styles.muted}>{hub.user.email || 'Visitor account'}</Text>
                    <div className={styles.statRow}>
                      {hub.permissions.map(permission => (
                        <Tag key={permission} color={roleColor(permission)}>
                          {permission}
                        </Tag>
                      ))}
                    </div>
                  </div>
                </div>
              </Card>

              <Card className={styles.panel} title="Profile">
                <Form form={profileForm} layout="vertical">
                  <Form.Item
                    name="displayName"
                    label="Chat username"
                    rules={[{ required: true, message: 'Chat username is required' }]}
                  >
                    <Input maxLength={30} />
                  </Form.Item>
                  <Form.Item name="profileImageUrl" label="Profile picture URL">
                    <Input maxLength={500} />
                  </Form.Item>
                  <div className={styles.uploadRow}>
                    <Upload
                      accept="image/png,image/jpeg,image/gif"
                      beforeUpload={file => {
                        uploadProfileImage(file as File);
                        return false;
                      }}
                      maxCount={1}
                      showUploadList={false}
                    >
                      <Button loading={uploading}>Upload picture</Button>
                    </Upload>
                    <Text className={styles.muted}>PNG, JPG, or GIF up to 2 MB.</Text>
                  </div>
                  <Button type="primary" loading={saving} onClick={saveProfile}>
                    Save profile
                  </Button>
                </Form>
              </Card>

              <Card className={styles.panel} title="Notifications">
                <Form form={notificationForm} layout="vertical">
                  <Form.Item name="email" valuePropName="checked">
                    <Checkbox>Email go-live alerts</Checkbox>
                  </Form.Item>
                  <Form.Item name="browserPush" valuePropName="checked">
                    <Checkbox>Browser push alerts</Checkbox>
                  </Form.Item>
                  <Form.Item name="messenger" valuePropName="checked">
                    <Checkbox>Facebook Messenger alerts</Checkbox>
                  </Form.Item>
                  <Form.Item
                    name="messengerDestination"
                    label="Messenger contact or page-scoped ID"
                  >
                    <Input maxLength={500} />
                  </Form.Item>
                  <Button type="primary" loading={saving} onClick={saveNotifications}>
                    Save notifications
                  </Button>
                </Form>
              </Card>

              <Card className={styles.panel} title="Reminder status">
                {hub.reminders?.length ? (
                  <div className={styles.scheduleList}>
                    {hub.reminders.map(reminder => (
                      <article className={styles.scheduleItem} key={reminder.id}>
                        <p className={styles.scheduleTime}>{formatDate(reminder.startsAt)}</p>
                        <p className={styles.scheduleTitle}>{reminder.title}</p>
                        <Text className={styles.muted}>
                          {[
                            reminder.email && 'email',
                            reminder.browserPush && 'browser push',
                            reminder.messenger && 'Messenger',
                          ]
                            .filter(Boolean)
                            .join(', ') || 'No channels'}
                        </Text>
                        <Tag color={reminderStatusColor(reminder)}>
                          {reminder.disabledAt
                            ? 'disabled'
                            : reminder.lastDeliveryStatus || 'scheduled'}
                        </Tag>
                        {reminder.lastDeliveryError && (
                          <Text className={styles.muted}>{reminder.lastDeliveryError}</Text>
                        )}
                      </article>
                    ))}
                  </div>
                ) : (
                  <Text className={styles.muted}>No saved schedule reminders yet.</Text>
                )}
              </Card>
            </section>

            <aside>
              <Card className={styles.panel} title="Destinations">
                <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                  {hub.destinations.map(destination => (
                    <div key={destination.key}>
                      <DestinationButton destination={destination} />
                      {!destination.available && destination.reason && (
                        <p className={styles.muted}>{destination.reason}</p>
                      )}
                    </div>
                  ))}
                </Space>
              </Card>

              <Card className={styles.panel} title="Stars wallet">
                <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                  <Statistic
                    title="Balance"
                    value={hub.stars?.wallet.balance || 0}
                    suffix="Stars"
                  />
                  <Statistic
                    title="Lifetime sent"
                    value={hub.stars?.wallet.lifetimeSent || 0}
                    suffix="Stars"
                  />
                  <Text className={styles.muted}>
                    Stars support the site and trigger live effects. They have no cash value and are
                    not paid out to streamers.
                  </Text>
                </Space>
              </Card>

              {hub.djProfile && (
                <Card className={styles.panel} title="Your DJ profile">
                  <p className={styles.djName}>{hub.djProfile.dj.displayName}</p>
                  <p className={styles.handle}>@{hub.djProfile.dj.handle}</p>
                  <ScheduleList
                    schedule={hub.djProfile.schedule}
                    accessToken={token}
                    reminderChannels={hub.notificationPreferences}
                  />
                </Card>
              )}

              <Card className={styles.panel} title="Upcoming lineup">
                <ScheduleList
                  schedule={hub.upcomingSchedule || []}
                  accessToken={token}
                  reminderChannels={hub.notificationPreferences}
                />
              </Card>
            </aside>
          </div>
        )}
      </div>
    </main>
  );
}

AccountPage.getLayout = function getLayout(page: ReactElement) {
  return page;
};
