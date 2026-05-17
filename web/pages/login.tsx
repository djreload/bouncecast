import React, { ReactElement, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import { Alert, Button, Card, Form, Input, Tabs, Typography, message } from 'antd';
import {
  BOUNCECAST_ADMIN_LOGIN,
  BOUNCECAST_STUDIO_LOGIN,
  fetchStudioData,
  getUnauthedData,
} from '../utils/apis';

const LoginOutlined = dynamic(() => import('@ant-design/icons/LoginOutlined'), { ssr: false });
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
  username?: string;
  login?: string;
  password: string;
};

export default function Login() {
  const [adminForm] = Form.useForm<LoginFormValues>();
  const [streamerForm] = Form.useForm<LoginFormValues>();
  const [saving, setSaving] = useState(false);
  const [hasStudioSession, setHasStudioSession] = useState(false);

  useEffect(() => {
    setHasStudioSession(Boolean(localStorage.getItem(studioTokenStorageKey)));
  }, []);

  const loginAdmin = async () => {
    const values = await adminForm.validateFields();
    setSaving(true);
    try {
      const result = await getUnauthedData(BOUNCECAST_ADMIN_LOGIN, {
        method: 'POST',
        data: {
          username: values.username || 'admin',
          password: values.password,
        },
      });
      window.location.assign(result.destination || '/admin/');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to log in as admin');
    } finally {
      setSaving(false);
    }
  };

  const loginStreamer = async () => {
    const values = await streamerForm.validateFields();
    setSaving(true);
    try {
      const result = await fetchStudioData(BOUNCECAST_STUDIO_LOGIN, undefined, {
        method: 'POST',
        data: {
          login: values.login,
          password: values.password,
        },
      });
      localStorage.setItem(studioTokenStorageKey, result.token);
      window.location.assign('/studio');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to log in as DJ');
    } finally {
      setSaving(false);
    }
  };

  return (
    <main className="bouncecast-studio-dashboard">
      <div className="studio-shell studio-login-shell">
        <section className="studio-live-strip">
          <div className="studio-live-copy">
            <h1>BounceCast Login</h1>
            <Text className="studio-brand-subtitle">
              One entry point for admins and DJs. Admins go to the control panel; DJs go to Studio.
            </Text>
          </div>
        </section>

        <section className="studio-login-panel">
          <div className="studio-brand-lockup">
            <img src="/logo" alt="BounceCast" />
            <div>
              <p className="studio-brand-title">Choose your dashboard</p>
              <Text className="studio-brand-subtitle">The right room opens after login.</Text>
            </div>
          </div>

          <Card className="studio-login-card">
            {hasStudioSession && (
              <Alert
                className="studio-register-alert"
                type="info"
                showIcon
                message="Studio session found"
                description="You can continue straight to the DJ dashboard, or log in again below."
                action={
                  <Button size="small" onClick={() => window.location.assign('/studio')}>
                    Open Studio
                  </Button>
                }
              />
            )}
            <Tabs
              defaultActiveKey="streamer"
              items={[
                {
                  key: 'streamer',
                  label: 'DJ / streamer',
                  children: (
                    <Form form={streamerForm} layout="vertical">
                      <Form.Item
                        name="login"
                        label="Handle or email"
                        rules={[{ required: true, message: 'Enter your handle or email' }]}
                      >
                        <Input
                          autoComplete="username"
                          prefix={<CustomerServiceOutlined />}
                          placeholder="@dj-name"
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
                        onClick={loginStreamer}
                      >
                        Log in to Studio
                      </Button>
                    </Form>
                  ),
                },
                {
                  key: 'admin',
                  label: 'Admin',
                  children: (
                    <Form form={adminForm} layout="vertical" initialValues={{ username: 'admin' }}>
                      <Form.Item
                        name="username"
                        label="Admin username"
                        rules={[{ required: true, message: 'Enter the admin username' }]}
                      >
                        <Input
                          autoComplete="username"
                          prefix={<SafetyCertificateOutlined />}
                          placeholder="admin"
                        />
                      </Form.Item>
                      <Form.Item
                        name="password"
                        label="Admin password"
                        rules={[{ required: true, message: 'Enter the admin password' }]}
                      >
                        <Input.Password autoComplete="current-password" />
                      </Form.Item>
                      <Button
                        type="primary"
                        block
                        icon={<LoginOutlined />}
                        loading={saving}
                        onClick={loginAdmin}
                      >
                        Log in to Admin
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
