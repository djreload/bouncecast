import React, { FC, useEffect, useState } from 'react';
import {
  Alert,
  Avatar,
  Button,
  Checkbox,
  Form,
  Input,
  Tabs,
  Tag,
  Typography,
  message as toast,
} from 'antd';
import { useRecoilState, useRecoilValue } from 'recoil';
import { AccountService, NotificationPreferencesPayload } from '../../../services/account-service';
import { registerWebPushNotifications } from '../../../services/notifications-service';
import {
  ACCESS_TOKEN_KEY,
  accessTokenAtom,
  clientConfigStateAtom,
  currentUserAtom,
} from '../../stores/ClientConfigStore';
import { CurrentUser } from '../../../interfaces/current-user';
import { setLocalStorage } from '../../../utils/localStorage';
import styles from './AccountModal.module.scss';

const { Text } = Typography;

type AccountModalProps = {
  closeModal: () => void;
};

const defaultNotificationPreferences = {
  email: false,
  browserPush: false,
  messenger: false,
  messengerDestination: '',
};

function userToCurrentUser(user): CurrentUser {
  const scopes = user?.scopes || [];
  return {
    id: user.id?.toString(),
    displayName: user.displayName,
    email: user.email,
    profileImageUrl: user.profileImageUrl,
    displayColor: user.displayColor,
    scopes,
    isModerator: scopes.includes('MODERATOR'),
    isOwner: scopes.includes('OWNER'),
    isAdmin: scopes.includes('ADMIN'),
    isDJ: scopes.includes('DJ'),
  };
}

function roleLabels(user?: CurrentUser): string[] {
  const roles = [];
  if (user?.isOwner) roles.push('owner');
  if (user?.isAdmin) roles.push('admin');
  if (user?.isModerator) roles.push('moderator');
  if (user?.isDJ) roles.push('dj');
  return roles.length > 0 ? roles : ['visitor'];
}

export const AccountModal: FC<AccountModalProps> = ({ closeModal }) => {
  const [accessToken, setAccessToken] = useRecoilState<string>(accessTokenAtom);
  const [currentUser, setCurrentUser] = useRecoilState<CurrentUser>(currentUserAtom);
  const clientConfig = useRecoilValue(clientConfigStateAtom);
  const [notificationPreferences, setNotificationPreferences] =
    useState<NotificationPreferencesPayload>(defaultNotificationPreferences);
  const [savingNotifications, setSavingNotifications] = useState(false);
  const [registerForm] = Form.useForm();
  const [loginForm] = Form.useForm();
  const [profileForm] = Form.useForm();
  const [notificationForm] = Form.useForm();

  const isRegistered = Boolean(currentUser?.email);

  const normalizeNotificationPreferences = preferences => ({
    ...defaultNotificationPreferences,
    ...(preferences || {}),
  });

  const applyAccountResponse = result => {
    if (result?.user) {
      setCurrentUser(userToCurrentUser(result.user));
    }
    if (result?.notificationPreferences) {
      setNotificationPreferences(normalizeNotificationPreferences(result.notificationPreferences));
    }

    const nextAccessToken = result?.accessToken;
    if (nextAccessToken) {
      setLocalStorage(ACCESS_TOKEN_KEY, nextAccessToken);
      setAccessToken(nextAccessToken);
      if (accessToken && accessToken !== nextAccessToken) {
        window.location.reload();
      }
    }
  };

  useEffect(() => {
    registerForm.setFieldsValue({
      displayName: currentUser?.displayName,
      profileImageUrl: currentUser?.profileImageUrl,
      notificationPreferences,
    });
    profileForm.setFieldsValue({
      displayName: currentUser?.displayName,
      profileImageUrl: currentUser?.profileImageUrl,
    });
    notificationForm.setFieldsValue(notificationPreferences);
  }, [currentUser, notificationPreferences]);

  useEffect(() => {
    if (!accessToken) {
      return;
    }

    AccountService.me(accessToken)
      .then(applyAccountResponse)
      .catch(() => {
        // Anonymous visitors can still register even if the current token is stale.
      });
  }, [accessToken]);

  const getBrowserPushEndpoint = async (enabled: boolean) => {
    if (!enabled) {
      return '';
    }
    const publicKey = clientConfig?.notifications?.browser?.publicKey;
    if (!clientConfig?.notifications?.browser?.enabled || !publicKey) {
      throw new Error('Browser push notifications are not enabled on this server yet');
    }
    return registerWebPushNotifications(publicKey);
  };

  const handleRegister = async () => {
    try {
      const values = await registerForm.validateFields();
      const result = await AccountService.register(accessToken, values);
      applyAccountResponse(result);
      toast.success('Account registered');
      closeModal();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to register account');
    }
  };

  const handleLogin = async () => {
    try {
      const values = await loginForm.validateFields();
      const result = await AccountService.login(values);
      applyAccountResponse(result);
      toast.success('Logged in');
      closeModal();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to log in');
    }
  };

  const handleProfileSave = async () => {
    try {
      const values = await profileForm.validateFields();
      const result = await AccountService.updateProfile(accessToken, values);
      applyAccountResponse(result);
      toast.success('Profile updated');
      closeModal();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to update profile');
    }
  };

  const handleNotificationSave = async () => {
    try {
      const values = await notificationForm.validateFields();
      setSavingNotifications(true);
      const payload = {
        ...values,
        browserPushEndpoint: await getBrowserPushEndpoint(Boolean(values.browserPush)),
      };
      const result = await AccountService.updateNotifications(accessToken, payload);
      applyAccountResponse(result);
      toast.success('Notification preferences updated');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to update notifications');
    } finally {
      setSavingNotifications(false);
    }
  };

  const notificationFields = (formPrefix?: string[], allowEmailOptIn = isRegistered) => (
    <div className={styles.notificationGrid}>
      <Form.Item
        name={formPrefix ? [...formPrefix, 'email'] : 'email'}
        valuePropName="checked"
        className={styles.notificationOption}
      >
        <Checkbox disabled={!allowEmailOptIn}>Email go-live alerts</Checkbox>
      </Form.Item>
      <Form.Item
        name={formPrefix ? [...formPrefix, 'browserPush'] : 'browserPush'}
        valuePropName="checked"
        className={styles.notificationOption}
      >
        <Checkbox>Browser push alerts</Checkbox>
      </Form.Item>
      <Form.Item
        name={formPrefix ? [...formPrefix, 'messenger'] : 'messenger'}
        valuePropName="checked"
        className={styles.notificationOption}
      >
        <Checkbox>Facebook Messenger alerts</Checkbox>
      </Form.Item>
      <Form.Item
        name={formPrefix ? [...formPrefix, 'messengerDestination'] : 'messengerDestination'}
        label="Messenger contact or page-scoped ID"
      >
        <Input placeholder="Messenger contact / PSID" maxLength={500} />
      </Form.Item>
    </div>
  );

  const accountHeader = (
    <div className={styles.accountHeader}>
      <Avatar src={currentUser?.profileImageUrl} size={54} className={styles.avatarPreview}>
        {currentUser?.displayName?.slice(0, 1)}
      </Avatar>
      <div>
        <p className={styles.accountTitle}>{currentUser?.displayName || 'Visitor'}</p>
        <p className={styles.accountMeta}>{currentUser?.email || 'Visitor chat account'}</p>
        <div className={styles.roleRow}>
          {roleLabels(currentUser).map(role => (
            <Tag key={role} color={role === 'visitor' ? 'default' : 'magenta'}>
              {role}
            </Tag>
          ))}
        </div>
      </div>
    </div>
  );

  const registerPanel = (
    <div className={styles.accountPanel}>
      {accountHeader}
      <Alert
        type="info"
        showIcon
        message="Visitor is the default role. Admins can approve owner, admin, moderator, and DJ permissions."
      />
      <Form form={registerForm} layout="vertical">
        <Form.Item
          name="displayName"
          label="Display name"
          rules={[{ required: true, message: 'Display name is required' }]}
        >
          <Input maxLength={30} />
        </Form.Item>
        <Form.Item
          name="email"
          label="Email"
          rules={[
            { required: true, message: 'Email is required' },
            { type: 'email', message: 'Enter a valid email address' },
          ]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          name="password"
          label="Password"
          rules={[
            { required: true, message: 'Password is required' },
            { min: 8, message: 'Password must be at least 8 characters' },
          ]}
        >
          <Input.Password />
        </Form.Item>
        <Form.Item name="profileImageUrl" label="Profile picture URL">
          <Input placeholder="https://example.com/avatar.png" maxLength={500} />
        </Form.Item>
        <div className={styles.notificationBox}>
          <Text strong>Go-live notifications</Text>
          <Text type="secondary">
            Opt in now and admins can use these preferences for future alerts.
          </Text>
          {notificationFields(['notificationPreferences'], true)}
        </div>
        <div className={styles.formActions}>
          <Button type="primary" onClick={handleRegister}>
            Register account
          </Button>
        </div>
      </Form>
    </div>
  );

  const loginPanel = (
    <div className={styles.accountPanel}>
      {accountHeader}
      <Form form={loginForm} layout="vertical">
        <Form.Item
          name="email"
          label="Email"
          rules={[
            { required: true, message: 'Email is required' },
            { type: 'email', message: 'Enter a valid email address' },
          ]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="password" label="Password" rules={[{ required: true }]}>
          <Input.Password />
        </Form.Item>
        <div className={styles.formActions}>
          <Button type="primary" onClick={handleLogin}>
            Log in
          </Button>
        </div>
      </Form>
    </div>
  );

  const profilePanel = (
    <div className={styles.accountPanel}>
      {accountHeader}
      <Text type="secondary">Your profile picture appears beside your chat messages.</Text>
      <Form form={profileForm} layout="vertical">
        <Form.Item
          name="displayName"
          label="Display name"
          rules={[{ required: true, message: 'Display name is required' }]}
        >
          <Input maxLength={30} />
        </Form.Item>
        <Form.Item name="profileImageUrl" label="Profile picture URL">
          <Input placeholder="https://example.com/avatar.png" maxLength={500} />
        </Form.Item>
        <div className={styles.formActions}>
          <Button type="primary" onClick={handleProfileSave}>
            Save profile
          </Button>
        </div>
      </Form>
    </div>
  );

  const notificationsPanel = (
    <div className={styles.accountPanel}>
      {accountHeader}
      <Alert
        type="info"
        showIcon
        message="Viewer notifications only"
        description="These opt-ins are for go-live alerts. They do not grant Admin or DJ dashboard access."
      />
      <Form form={notificationForm} layout="vertical" initialValues={defaultNotificationPreferences}>
        {notificationFields()}
        {!isRegistered && (
          <Alert
            type="warning"
            showIcon
            message="Register first to use email alerts"
            description="Browser push can still be enabled from this device. Email alerts need a saved account email."
          />
        )}
        <div className={styles.formActions}>
          <Button type="primary" loading={savingNotifications} onClick={handleNotificationSave}>
            Save notifications
          </Button>
        </div>
      </Form>
    </div>
  );

  return (
    <Tabs
      defaultActiveKey={isRegistered ? 'profile' : 'register'}
      items={[
        { key: 'profile', label: 'Profile', children: profilePanel },
        { key: 'notifications', label: 'Notifications', children: notificationsPanel },
        { key: 'register', label: 'Register', children: registerPanel },
        { key: 'login', label: 'Log in', children: loginPanel },
      ]}
    />
  );
};
