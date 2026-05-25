export type MobileAppSettings = {
  name: string;
  publicBaseUrl?: string;
  maintenance_mode?: boolean;
  maintenance_message?: string;
  minimum_supported_version: string;
  recommended_version?: string;
  force_update?: boolean;
  force_update_message?: string;
  homepage_message?: string;
  support_url?: string;
};

export type MobileBrandingSettings = {
  logo_url?: string;
  splash_url?: string;
  app_icon_url?: string;
  app_background_url?: string;
  primary_color: string;
  accent_color: string;
  background_color: string;
  theme_mode: 'system' | 'dark' | 'light';
};

export type MobileFeatureFlags = {
  chat: boolean;
  gif_picker: boolean;
  stickers: boolean;
  profiles: boolean;
  stars: boolean;
  chat_reactions: boolean;
  stars_overlay: boolean;
  reward_wheel: boolean;
  reward_overlay: boolean;
  push_notifications: boolean;
  ads: boolean;
  experimental_features: boolean;
};

export type MobileAdSettings = {
  enabled: boolean;
  test_mode: boolean;
  fallback_enabled: boolean;
  priority: string[];
  google_enabled: boolean;
  unity_enabled: boolean;
  google_app_id?: string;
  google_banner_ad_unit_id?: string;
  google_app_open_ad_unit_id?: string;
  unity_game_id?: string;
  unity_banner_placement_id?: string;
  unity_app_open_placement_id?: string;
  banner_enabled: boolean;
  banner_position: 'bottom' | 'top';
  app_open_enabled: boolean;
  app_open_cooldown_minutes: number;
  app_open_show_on_first_launch: boolean;
  timeout_ms: number;
  retry_limit: number;
  consent_required: boolean;
  personalized_ads_allowed: boolean;
};

export type MobileNotificationSettings = {
  go_live_enabled: boolean;
  title_template: string;
  body_template: string;
  image_url?: string;
  icon_url?: string;
  topic: string;
  fcm_project_id?: string;
  last_successful_send_at?: string;
  last_error?: string;
};

export type MobileNavigationItem = {
  id?: number;
  label: string;
  url: string;
  item_type: 'internal' | 'external';
  display_order: number;
  enabled: boolean;
};

export type MobileLegalPage = {
  id?: number;
  slug: string;
  title: string;
  content?: string;
  url?: string;
  display_order: number;
  enabled: boolean;
};

export type MobileAdminSettings = {
  app: MobileAppSettings;
  branding: MobileBrandingSettings;
  features: MobileFeatureFlags;
  ads: MobileAdSettings;
  notifications: MobileNotificationSettings;
  navigation: MobileNavigationItem[];
  legal: MobileLegalPage[];
  deviceCount: number;
};
