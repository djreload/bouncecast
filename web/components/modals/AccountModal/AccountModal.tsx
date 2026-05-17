import { FC, useEffect } from 'react';
import { Alert, Avatar, Button, Form, Input, Tabs, Tag, Typography, message as toast } from 'antd';
import { useRecoilState } from 'recoil';
import { AccountService } from '../../../services/account-service';
import { ACCESS_TOKEN_KEY, accessTokenAtom, currentUserAtom } from '../../stores/ClientConfigStore';
import { CurrentUser } from '../../../interfaces/current-user';
import { setLocalStorage } from '../../../utils/localStorage';
import styles from './AccountModal.module.scss';

const { Text } = Typography;

type AccountModalProps = {
  closeModal: () => void;
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
  const [registerForm] = Form.useForm();
  const [loginForm] = Form.useForm();
  const [profileForm] = Form.useForm();

  const isRegistered = Boolean(currentUser?.email);

  useEffect(() => {
    registerForm.setFieldsValue({
      displayName: currentUser?.displayName,
      profileImageUrl: currentUser?.profileImageUrl,
    });
    profileForm.setFieldsValue({
      displayName: currentUser?.displayName,
      profileImageUrl: currentUser?.profileImageUrl,
    });
  }, [currentUser]);

  const applyAccountResponse = result => {
    if (result?.user) {
      setCurrentUser(userToCurrentUser(result.user));
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

  return (
    <Tabs
      defaultActiveKey={isRegistered ? 'profile' : 'register'}
      items={[
        { key: 'profile', label: 'Profile', children: profilePanel },
        { key: 'register', label: 'Register', children: registerPanel },
        { key: 'login', label: 'Log in', children: loginPanel },
      ]}
    />
  );
};
