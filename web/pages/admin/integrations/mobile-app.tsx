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
  Select,
  Space,
  Statistic,
  Switch,
  Tabs,
  Typography,
  Upload,
  message,
} from 'antd';
import { RcFile } from 'antd/lib/upload/interface';
import dynamic from 'next/dynamic';
import { AdminLayout } from '../../../components/layouts/AdminLayout';
import {
  BOUNCECAST_MOBILE_ADMIN,
  BOUNCECAST_MOBILE_ASSET_UPLOAD,
  BOUNCECAST_MOBILE_SETTINGS,
  fetchData,
  postAdminFormData,
} from '../../../utils/apis';
import {
  MobileAdminSettings,
  MobileLegalPage,
  MobileNavigationItem,
} from '../../../interfaces/mobile.model';
import { ACCEPTED_IMAGE_TYPES, readableBytes } from '../../../utils/images';

const MobileOutlined = dynamic(() => import('@ant-design/icons/MobileOutlined'), { ssr: false });
const CopyOutlined = dynamic(() => import('@ant-design/icons/CopyOutlined'), { ssr: false });
const UploadOutlined = dynamic(() => import('@ant-design/icons/UploadOutlined'), { ssr: false });

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const MAX_MOBILE_ASSET_FILESIZE = 5 * 1024 * 1024;

const defaultMobileSettings: MobileAdminSettings = {
  app: {
    name: 'BounceCast',
    minimum_supported_version: '1.0.0',
  },
  branding: {
    primary_color: '#080711',
    accent_color: '#ff2a8a',
    background_color: '#05050f',
    theme_mode: 'system',
  },
  features: {
    chat: true,
    gif_picker: true,
    stickers: true,
    profiles: true,
    stars: true,
    chat_reactions: true,
    stars_overlay: true,
    reward_wheel: true,
    reward_overlay: true,
    push_notifications: false,
    ads: false,
    experimental_features: false,
  },
  ads: {
    enabled: false,
    test_mode: true,
    fallback_enabled: true,
    priority: ['google', 'unity'],
    google_enabled: false,
    unity_enabled: false,
    banner_enabled: false,
    banner_position: 'bottom',
    app_open_enabled: false,
    app_open_cooldown_minutes: 30,
    app_open_show_on_first_launch: true,
    timeout_ms: 5000,
    retry_limit: 1,
    consent_required: true,
    personalized_ads_allowed: false,
  },
  notifications: {
    go_live_enabled: false,
    title_template: 'BounceCast is live',
    body_template: '{stream_title} is live now. Tap to watch.',
    topic: 'go-live',
  },
  navigation: [
    { label: 'Live', url: '/', item_type: 'internal', display_order: 10, enabled: true },
    {
      label: 'Schedule',
      url: '/schedule',
      item_type: 'internal',
      display_order: 20,
      enabled: true,
    },
    { label: 'DJs', url: '/djs', item_type: 'internal', display_order: 30, enabled: true },
  ],
  legal: [
    { slug: 'privacy', title: 'Privacy Policy', display_order: 10, enabled: false },
    { slug: 'terms', title: 'Terms and Conditions', display_order: 20, enabled: false },
  ],
  deviceCount: 0,
};

function prettyJSON(value: unknown) {
  return JSON.stringify(value || [], null, 2);
}

function parseJSONList<T>(value: string, label: string): T[] {
  try {
    const parsed = JSON.parse(value || '[]');
    if (!Array.isArray(parsed)) throw new Error(`${label} must be a JSON array`);
    return parsed;
  } catch (error) {
    throw new Error(error instanceof Error ? error.message : `${label} JSON is invalid`);
  }
}

function MobileAssetUploadButton({
  form,
  fieldName,
  assetType,
}: {
  form: any;
  fieldName: keyof MobileAdminSettings['branding'];
  assetType: string;
}) {
  const [uploading, setUploading] = useState(false);

  const beforeUpload = (file: RcFile) => {
    if (file.size > MAX_MOBILE_ASSET_FILESIZE) {
      message.error(`Image is too large (${readableBytes(file.size)}). Maximum size is 5 MB.`);
      return Upload.LIST_IGNORE;
    }
    if (!ACCEPTED_IMAGE_TYPES.includes(file.type)) {
      message.error(`Unsupported image type: ${file.type}`);
      return Upload.LIST_IGNORE;
    }
    return true;
  };

  const uploadAsset = async ({ file, onError, onSuccess }: any) => {
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('asset', file as RcFile);
      formData.append('type', assetType);
      const result = await postAdminFormData(
        `${BOUNCECAST_MOBILE_ASSET_UPLOAD}?type=${encodeURIComponent(assetType)}`,
        formData,
      );
      const branding = form.getFieldValue('branding') || {};
      form.setFieldsValue({
        branding: {
          ...branding,
          [fieldName]: result.url,
        },
      });
      message.success('Mobile asset uploaded');
      onSuccess?.(result);
    } catch (error) {
      const uploadError = error instanceof Error ? error : new Error('Unable to upload image');
      message.error(uploadError.message);
      onError?.(uploadError);
    } finally {
      setUploading(false);
    }
  };

  return (
    <Upload
      accept={ACCEPTED_IMAGE_TYPES.join(',')}
      beforeUpload={beforeUpload}
      customRequest={uploadAsset}
      showUploadList={false}
      disabled={uploading}
    >
      <Button icon={<UploadOutlined />} loading={uploading}>
        Upload image
      </Button>
    </Upload>
  );
}

export default function MobileAppAdmin() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [settings, setSettings] = useState<MobileAdminSettings | null>(null);
  const [navigationJSON, setNavigationJSON] = useState(
    prettyJSON(defaultMobileSettings.navigation),
  );
  const [legalJSON, setLegalJSON] = useState(prettyJSON(defaultMobileSettings.legal));
  const [form] = Form.useForm<MobileAdminSettings>();

  const configURL = useMemo(() => {
    if (typeof window === 'undefined') return '/api/mobile/v1/config';
    return `${window.location.origin}/api/mobile/v1/config`;
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const result = await fetchData(BOUNCECAST_MOBILE_ADMIN);
      const merged = {
        ...defaultMobileSettings,
        ...result,
        app: { ...defaultMobileSettings.app, ...(result?.app || {}) },
        branding: { ...defaultMobileSettings.branding, ...(result?.branding || {}) },
        features: { ...defaultMobileSettings.features, ...(result?.features || {}) },
        ads: { ...defaultMobileSettings.ads, ...(result?.ads || {}) },
        notifications: {
          ...defaultMobileSettings.notifications,
          ...(result?.notifications || {}),
        },
      };
      setSettings(merged);
      form.setFieldsValue(merged);
      setNavigationJSON(prettyJSON(merged.navigation));
      setLegalJSON(prettyJSON(merged.legal));
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load mobile app settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const saveSettings = async () => {
    setSaving(true);
    try {
      const values = await form.validateFields();
      const payload: MobileAdminSettings = {
        ...defaultMobileSettings,
        ...settings,
        ...values,
        navigation: parseJSONList<MobileNavigationItem>(navigationJSON, 'Navigation'),
        legal: parseJSONList<MobileLegalPage>(legalJSON, 'Legal pages'),
      };
      const result = await fetchData(BOUNCECAST_MOBILE_SETTINGS, {
        method: 'POST',
        data: payload,
      });
      setSettings(result);
      setNavigationJSON(prettyJSON(result.navigation));
      setLegalJSON(prettyJSON(result.legal));
      form.setFieldsValue(result);
      message.success('Mobile app settings saved');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save mobile app settings');
    } finally {
      setSaving(false);
    }
  };

  const copyConfigURL = async () => {
    await navigator.clipboard?.writeText(configURL);
    message.success('Mobile config URL copied');
  };

  return (
    <div className="bouncecast-admin-page">
      <Space align="center" size="middle" wrap>
        <MobileOutlined style={{ fontSize: 28 }} />
        <div>
          <Title level={2} style={{ marginBottom: 0 }}>
            Mobile App Platform
          </Title>
          <Text type="secondary">
            Configure the official BounceCast Android client without creating a separate stream or
            chat system.
          </Text>
        </div>
      </Space>

      <Row gutter={[16, 16]} style={{ marginTop: 20 }}>
        <Col xs={24} md={8}>
          <Card className="studio-panel">
            <Statistic title="Registered mobile devices" value={settings?.deviceCount || 0} />
          </Card>
        </Col>
        <Col xs={24} md={16}>
          <Card className="studio-panel" title="Public mobile API">
            <Paragraph copyable={{ text: configURL }} style={{ marginBottom: 8 }}>
              <Text code>{configURL}</Text>
            </Paragraph>
            <Button icon={<CopyOutlined />} onClick={copyConfigURL}>
              Copy config URL
            </Button>
          </Card>
        </Col>
      </Row>

      <Alert
        type="info"
        showIcon
        style={{ margin: '20px 0' }}
        message="Same stream, same chat"
        description="The mobile API points the Android app at /hls/stream.m3u8, /ws, /api/status, and the existing BounceCast user/chat APIs. Mobile users are another official client, not a second community."
      />

      <Card className="studio-panel" loading={loading}>
        <Form form={form} layout="vertical" disabled={saving}>
          <Tabs
            items={[
              {
                key: 'app',
                label: 'App',
                children: (
                  <Row gutter={16}>
                    <Col xs={24} md={12}>
                      <Form.Item name={['app', 'name']} label="App display name">
                        <Input maxLength={80} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name={['app', 'publicBaseUrl']} label="Public BounceCast URL">
                        <Input placeholder="https://k-nrg.co.uk" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item
                        name={['app', 'minimum_supported_version']}
                        label="Minimum app version"
                      >
                        <Input placeholder="1.0.0" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item
                        name={['app', 'recommended_version']}
                        label="Recommended app version"
                      >
                        <Input placeholder="1.0.1" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name={['app', 'support_url']} label="Support URL">
                        <Input placeholder="https://example.com/support" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['app', 'maintenance_mode']}
                        label="Maintenance mode"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['app', 'force_update']}
                        label="Force update"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24}>
                      <Form.Item name={['app', 'homepage_message']} label="Homepage message">
                        <TextArea rows={3} maxLength={500} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name={['app', 'maintenance_message']} label="Maintenance message">
                        <TextArea rows={3} maxLength={280} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['app', 'force_update_message']}
                        label="Force update message"
                      >
                        <TextArea rows={3} maxLength={280} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'branding',
                label: 'Branding',
                children: (
                  <Row gutter={16}>
                    <Col xs={24} md={8}>
                      <Form.Item name={['branding', 'logo_url']} label="Logo URL">
                        <Input placeholder="/logo" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name={['branding', 'splash_url']} label="Splash image URL">
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name={['branding', 'app_icon_url']} label="App icon URL">
                        <Input placeholder="/favicon.ico" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['branding', 'app_background_url']}
                        label="App background image URL"
                        extra="Used behind the native mobile livestream, offline screen, and mobile panels."
                      >
                        <Input placeholder="/public/mobile-assets/app-background.jpg" />
                      </Form.Item>
                      <MobileAssetUploadButton
                        form={form}
                        fieldName="app_background_url"
                        assetType="app-background"
                      />
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['branding', 'primary_color']} label="Primary color">
                        <Input type="color" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['branding', 'accent_color']} label="Accent color">
                        <Input type="color" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['branding', 'background_color']} label="Background color">
                        <Input type="color" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['branding', 'theme_mode']} label="Theme mode">
                        <Select
                          options={[
                            { value: 'system', label: 'System' },
                            { value: 'dark', label: 'Dark' },
                            { value: 'light', label: 'Light' },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'features',
                label: 'Features',
                children: (
                  <Row gutter={[16, 16]}>
                    {[
                      ['chat', 'Chat'],
                      ['gif_picker', 'GIF picker'],
                      ['stickers', 'Stickers'],
                      ['profiles', 'Profiles'],
                      ['stars', 'Stars'],
                      ['chat_reactions', 'Chat reactions'],
                      ['stars_overlay', 'Stars overlays'],
                      ['reward_wheel', 'Rewards Wheel'],
                      ['reward_overlay', 'Reward overlays'],
                      ['push_notifications', 'Push notifications'],
                      ['ads', 'Ads'],
                      ['experimental_features', 'Experimental features'],
                    ].map(([key, label]) => (
                      <Col xs={24} md={8} key={key}>
                        <Form.Item name={['features', key]} label={label} valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Col>
                    ))}
                  </Row>
                ),
              },
              {
                key: 'ads',
                label: 'Ads',
                children: (
                  <Row gutter={16}>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'enabled']}
                        label="Ads enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'test_mode']}
                        label="Test mode"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'fallback_enabled']}
                        label="Fallback enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['ads', 'priority']} label="Provider priority">
                        <Select
                          mode="multiple"
                          options={[
                            { value: 'google', label: 'Google' },
                            { value: 'unity', label: 'Unity' },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'google_enabled']}
                        label="Google enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'unity_enabled']}
                        label="Unity enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name={['ads', 'google_app_id']} label="Google app ID">
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['ads', 'google_banner_ad_unit_id']}
                        label="Google banner ad unit ID"
                      >
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['ads', 'google_app_open_ad_unit_id']}
                        label="Google app-open ad unit ID"
                      >
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name={['ads', 'unity_game_id']} label="Unity game ID">
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['ads', 'unity_banner_placement_id']}
                        label="Unity banner placement ID"
                      >
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['ads', 'unity_app_open_placement_id']}
                        label="Unity interstitial/app-open placement ID"
                      >
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'banner_enabled']}
                        label="Banner enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['ads', 'banner_position']} label="Banner position">
                        <Select
                          options={[
                            { value: 'bottom', label: 'Bottom' },
                            { value: 'top', label: 'Top' },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'app_open_enabled']}
                        label="App-open enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'app_open_cooldown_minutes']}
                        label="App-open cooldown"
                      >
                        <InputNumber min={1} max={1440} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['ads', 'timeout_ms']} label="Timeout ms">
                        <InputNumber min={1000} max={30000} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={['ads', 'retry_limit']} label="Retry limit">
                        <InputNumber min={0} max={5} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'app_open_show_on_first_launch']}
                        label="Show on first launch"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'consent_required']}
                        label="Consent required"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        name={['ads', 'personalized_ads_allowed']}
                        label="Personalized ads"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'notifications',
                label: 'Push',
                children: (
                  <Row gutter={16}>
                    <Col xs={24} md={8}>
                      <Form.Item
                        name={['notifications', 'go_live_enabled']}
                        label="Go-live push enabled"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name={['notifications', 'topic']} label="FCM topic">
                        <Input placeholder="go-live" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name={['notifications', 'fcm_project_id']} label="FCM project ID">
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name={['notifications', 'title_template']} label="Title template">
                        <Input maxLength={120} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item
                        name={['notifications', 'image_url']}
                        label="Notification image URL"
                      >
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24}>
                      <Form.Item name={['notifications', 'body_template']} label="Body template">
                        <TextArea rows={3} maxLength={500} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'navigation',
                label: 'Navigation & legal',
                children: (
                  <Row gutter={16}>
                    <Col xs={24} md={12}>
                      <Typography.Title level={4}>Navigation JSON</Typography.Title>
                      <TextArea
                        rows={14}
                        value={navigationJSON}
                        onChange={event => setNavigationJSON(event.target.value)}
                      />
                    </Col>
                    <Col xs={24} md={12}>
                      <Typography.Title level={4}>Legal pages JSON</Typography.Title>
                      <TextArea
                        rows={14}
                        value={legalJSON}
                        onChange={event => setLegalJSON(event.target.value)}
                      />
                    </Col>
                  </Row>
                ),
              },
            ]}
          />

          <Space style={{ marginTop: 20 }}>
            <Button type="primary" onClick={saveSettings} loading={saving}>
              Save mobile settings
            </Button>
            <Button onClick={loadSettings} disabled={saving}>
              Reload
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  );
}

MobileAppAdmin.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
