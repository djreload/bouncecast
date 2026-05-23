import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import dynamic from 'next/dynamic';
import { Alert, Button, Card, Checkbox, Form, Input, Tabs, Tag, Typography, message } from 'antd';
import {
  BOUNCECAST_AUTH_LOGIN,
  BOUNCECAST_STUDIO_REGISTER,
  fetchStudioData,
  getUnauthedData,
} from '../utils/apis';
import { AccountService } from '../services/account-service';
import { ACCESS_TOKEN_KEY } from '../components/stores/ClientConfigStore';

const LoginOutlined = dynamic(() => import('@ant-design/icons/LoginOutlined'), { ssr: false });
const UserAddOutlined = dynamic(() => import('@ant-design/icons/UserAddOutlined'), { ssr: false });
const CustomerServiceOutlined = dynamic(() => import('@ant-design/icons/CustomerServiceOutlined'), {
  ssr: false,
});
const SafetyCertificateOutlined = dynamic(
  () => import('@ant-design/icons/SafetyCertificateOutlined'),
  { ssr: false },
);

const { Text } = Typography;
const studioTokenStorageKey = 'bouncecastStudioToken';

type LoginFormValues = {
  login: string;
  password: string;
};

type RegisterFormValues = {
  displayName: string;
  email: string;
  password: string;
  profileImageUrl?: string;
  notificationPreferences?: {
    email?: boolean;
  };
};

type DJRegisterFormValues = {
  displayName: string;
  handle: string;
  email: string;
  password: string;
};

type UnifiedLoginResult = {
  accessToken?: string;
  studioToken?: string;
  permissions?: string[];
  destination?: string;
  message?: string;
};

function safeNext(value: string | string[] | undefined): string {
  const next = Array.isArray(value) ? value[0] : value || '';
  if (!next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) {
    return '';
  }
  return next;
}

function storeUnifiedLogin(result: UnifiedLoginResult) {
  if (result?.accessToken) {
    localStorage.setItem(ACCESS_TOKEN_KEY, result.accessToken);
  }
  if (result?.studioToken) {
    localStorage.setItem(studioTokenStorageKey, result.studioToken);
  }
}

function roleTags(permissions?: string[]) {
  const list = permissions?.length ? permissions : ['user'];
  return (
    <div className="studio-session-meta">
      {list.map(permission => (
        <Tag key={permission} color={permission === 'owner' ? 'gold' : 'cyan'}>
          {permission}
        </Tag>
      ))}
    </div>
  );
}

export default function Login() {
  const router = useRouter();
  const next = useMemo(() => safeNext(router.query.next), [router.query.next]);
  const requestedMode = String(router.query.mode || '');
  const [loginForm] = Form.useForm<LoginFormValues>();
  const [registerForm] = Form.useForm<RegisterFormValues>();
  const [djRegisterForm] = Form.useForm<DJRegisterFormValues>();
  const [saving, setSaving] = useState(false);
  const [existingSession, setExistingSession] = useState({ account: false, studio: false });
  const [lastLogin, setLastLogin] = useState<UnifiedLoginResult | null>(null);
  const [registrationSubmitted, setRegistrationSubmitted] = useState(false);

  useEffect(() => {
    setExistingSession({
      account: Boolean(localStorage.getItem(ACCESS_TOKEN_KEY)),
      studio: Boolean(localStorage.getItem(studioTokenStorageKey)),
    });
  }, []);

  const finishLogin = (result: UnifiedLoginResult) => {
    storeUnifiedLogin(result);
    setLastLogin(result);
    message.success(result?.message || 'Logged in');
    window.location.assign(result?.destination || next || '/account');
  };

  const login = async () => {
    const values = await loginForm.validateFields();
    setSaving(true);
    try {
      const result = await getUnauthedData(BOUNCECAST_AUTH_LOGIN, {
        method: 'POST',
        data: {
          login: values.login,
          password: values.password,
          next,
        },
      });
      finishLogin(result);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to log in');
    } finally {
      setSaving(false);
    }
  };

  const registerAccount = async () => {
    const values = await registerForm.validateFields();
    setSaving(true);
    try {
      const result = await AccountService.register('', values);
      storeUnifiedLogin(result);
      message.success('Account registered');
      window.location.assign(next || '/account');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to register account');
    } finally {
      setSaving(false);
    }
  };

  const requestDJAccess = async () => {
    const values = await djRegisterForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchStudioData(BOUNCECAST_STUDIO_REGISTER, undefined, {
        method: 'POST',
        data: values,
      });
      djRegisterForm.resetFields();
      setRegistrationSubmitted(true);
      message.success(result.message || 'DJ request received');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to request DJ access');
    } finally {
      setSaving(false);
    }
  };

  const defaultTab = requestedMode === 'dj-register' ? 'dj-register' : 'login';

  return (
    <main className="bouncecast-studio-dashboard">
      <div className="studio-shell studio-login-shell unified-login-shell">
        <section className="studio-live-strip unified-login-hero">
          <div className="studio-live-copy">
            <p className="studio-eyebrow">Unified access</p>
            <h1>One BounceCast login</h1>
            <Text className="studio-brand-subtitle">
              Sign in once for chat, Stars, account settings, DJ Studio, and admin tools. Your role
              decides which rooms open.
            </Text>
            <div className="unified-login-links">
              <Link href="/">Live stream</Link>
              <Link href="/account">Account Hub</Link>
              <Link href="/djs">DJ lineup</Link>
            </div>
          </div>
        </section>

        <section className="studio-login-panel">
          <div className="studio-brand-lockup">
            <img src="/logo" alt="BounceCast" />
            <div>
              <p className="studio-brand-title">BounceCast Access</p>
              <Text className="studio-brand-subtitle">
                Admin, DJ, moderator, and viewer accounts all start here.
              </Text>
            </div>
          </div>

          <Card className="studio-login-card">
            {(existingSession.account || existingSession.studio) && (
              <Alert
                className="studio-register-alert"
                type="info"
                showIcon
                message="Session found"
                description="You can open your account area or log in again to refresh permissions."
                action={
                  <Button size="small" onClick={() => window.location.assign('/account')}>
                    Account Hub
                  </Button>
                }
              />
            )}

            {registrationSubmitted && (
              <Alert
                className="studio-register-alert"
                type="success"
                showIcon
                message="DJ request sent"
                description="Your account stays inactive until an admin activates DJ access."
              />
            )}

            {lastLogin && (
              <Alert
                className="studio-register-alert"
                type="success"
                showIcon
                message="Role matched"
                description={roleTags(lastLogin.permissions)}
              />
            )}

            <Tabs
              defaultActiveKey={defaultTab}
              items={[
                {
                  key: 'login',
                  label: 'Log in',
                  children: (
                    <Form form={loginForm} layout="vertical">
                      <Form.Item
                        name="login"
                        label="Email, DJ handle, or admin username"
                        rules={[{ required: true, message: 'Enter your login' }]}
                      >
                        <Input
                          autoComplete="username"
                          prefix={<SafetyCertificateOutlined />}
                          placeholder="admin, name@example.com, or @dj-name"
                        />
                      </Form.Item>
                      <Form.Item
                        name="password"
                        label="Password"
                        rules={[{ required: true, message: 'Enter your password' }]}
                      >
                        <Input.Password autoComplete="current-password" />
                      </Form.Item>
                      <Button
                        type="primary"
                        block
                        icon={<LoginOutlined />}
                        loading={saving}
                        onClick={login}
                      >
                        Log in
                      </Button>
                    </Form>
                  ),
                },
                {
                  key: 'register',
                  label: 'Create account',
                  children: (
                    <Form form={registerForm} layout="vertical">
                      <Form.Item
                        name="displayName"
                        label="Chat username"
                        rules={[{ required: true, message: 'Choose your chat username' }]}
                      >
                        <Input maxLength={30} autoComplete="nickname" />
                      </Form.Item>
                      <Form.Item
                        name="email"
                        label="Email"
                        rules={[
                          { required: true, message: 'Email is required' },
                          { type: 'email', message: 'Use a valid email address' },
                        ]}
                      >
                        <Input autoComplete="email" />
                      </Form.Item>
                      <Form.Item
                        name="password"
                        label="Password"
                        rules={[
                          { required: true, message: 'Password is required' },
                          { min: 8, message: 'Use at least 8 characters' },
                        ]}
                      >
                        <Input.Password autoComplete="new-password" />
                      </Form.Item>
                      <Form.Item name="profileImageUrl" label="Profile picture URL">
                        <Input maxLength={500} placeholder="/public/profiles/avatar.png" />
                      </Form.Item>
                      <Form.Item
                        name={['notificationPreferences', 'email']}
                        valuePropName="checked"
                      >
                        <Checkbox>Email go-live alerts</Checkbox>
                      </Form.Item>
                      <Button
                        type="primary"
                        block
                        icon={<UserAddOutlined />}
                        loading={saving}
                        onClick={registerAccount}
                      >
                        Create account
                      </Button>
                    </Form>
                  ),
                },
                {
                  key: 'dj-register',
                  label: 'DJ access',
                  children: (
                    <Form form={djRegisterForm} layout="vertical">
                      <Form.Item
                        name="displayName"
                        label="DJ name"
                        rules={[{ required: true, message: 'Add your DJ name' }]}
                      >
                        <Input
                          maxLength={80}
                          autoComplete="name"
                          prefix={<CustomerServiceOutlined />}
                        />
                      </Form.Item>
                      <Form.Item
                        name="handle"
                        label="Handle"
                        rules={[{ required: true, message: 'Choose a handle' }]}
                      >
                        <Input maxLength={32} autoComplete="username" placeholder="dj-name" />
                      </Form.Item>
                      <Form.Item
                        name="email"
                        label="Email"
                        rules={[
                          { required: true, message: 'Email is required' },
                          { type: 'email', message: 'Use a valid email address' },
                        ]}
                      >
                        <Input autoComplete="email" />
                      </Form.Item>
                      <Form.Item
                        name="password"
                        label="Password"
                        rules={[
                          { required: true, message: 'Password is required' },
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
                        onClick={requestDJAccess}
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

Login.getLayout = function getLayout(page: ReactElement) {
  return page;
};
