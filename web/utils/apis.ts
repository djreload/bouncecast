const semverGt = require('semver/functions/gt');

/* eslint-disable prefer-destructuring */
const ADMIN_USERNAME = process.env.NEXT_PUBLIC_ADMIN_USERNAME;
const ADMIN_STREAMKEY = process.env.NEXT_PUBLIC_ADMIN_STREAMKEY;
export const NEXT_PUBLIC_API_HOST = process.env.NEXT_PUBLIC_API_HOST;

const API_LOCATION = `${NEXT_PUBLIC_API_HOST}api/admin/`;

export const FETCH_INTERVAL = 15000;

// Current inbound broadcaster info
export const STATUS = `${API_LOCATION}status`;

// Current server config
export const SERVER_CONFIG = `${API_LOCATION}serverconfig`;

// Base url to update config settings
export const SERVER_CONFIG_UPDATE_URL = `${API_LOCATION}config`;

// Get viewer count over time
export const VIEWERS_OVER_TIME = `${API_LOCATION}viewersOverTime`;

// Get active viewer details
export const ACTIVE_VIEWER_DETAILS = `${API_LOCATION}viewers`;

// Get currently connected chat clients
export const CONNECTED_CLIENTS = `${API_LOCATION}chat/clients`;

// Get list of disabled/blocked chat users
export const DISABLED_USERS = `${API_LOCATION}chat/users/disabled`;

// Disable/enable a single user
export const USER_ENABLED_TOGGLE = `${API_LOCATION}chat/users/setenabled`;

// Get banned IP addresses
export const BANNED_IPS = `${API_LOCATION}chat/users/ipbans`;

// Remove IP ban
export const BANNED_IP_REMOVE = `${API_LOCATION}chat/users/ipbans/remove`;

// Disable/enable a single user
export const USER_SET_MODERATOR = `${API_LOCATION}chat/users/setmoderator`;

// Get list of moderators
export const MODERATORS = `${API_LOCATION}chat/users/moderators`;

// Get hardware stats
export const HARDWARE_STATS = `${API_LOCATION}hardwarestats`;

// Get all logs
export const LOGS_ALL = `${API_LOCATION}logs`;

// Get warnings + errors
export const LOGS_WARN = `${API_LOCATION}logs/warnings`;

// Get chat history
export const CHAT_HISTORY = `${API_LOCATION}chat/messages`;

// Get chat history
export const UPDATE_CHAT_MESSGAE_VIZ = `/api/admin/chat/messagevisibility`;

// Upload a new custom emoji
export const UPLOAD_EMOJI = `${API_LOCATION}emoji/upload`;

// Delete a custom emoji
export const DELETE_EMOJI = `${API_LOCATION}emoji/delete`;

// Get all access tokens
export const ACCESS_TOKENS = `${API_LOCATION}accesstokens`;

// Delete a single access token
export const DELETE_ACCESS_TOKEN = `${API_LOCATION}accesstokens/delete`;

// Create a new access token
export const CREATE_ACCESS_TOKEN = `${API_LOCATION}accesstokens/create`;

// Get webhooks
export const WEBHOOKS = `${API_LOCATION}webhooks`;

// Delete a single webhook
export const DELETE_WEBHOOK = `${API_LOCATION}webhooks/delete`;

// Create a single webhook
export const CREATE_WEBHOOK = `${API_LOCATION}webhooks/create`;

// hard coded social icons list
export const SOCIAL_PLATFORMS_LIST = `${NEXT_PUBLIC_API_HOST}api/socialplatforms`;

// send a message to the fediverse
export const FEDERATION_MESSAGE_SEND = `${API_LOCATION}federation/send`;

// Get followers
export const FOLLOWERS = `${API_LOCATION}followers`;

// Get followers pending approval
export const FOLLOWERS_PENDING = `${API_LOCATION}followers/pending`;

// Get followers who were blocked or rejected
export const FOLLOWERS_BLOCKED = `${API_LOCATION}followers/blocked`;

// Approve, reject a follow request
export const SET_FOLLOWER_APPROVAL = `${API_LOCATION}followers/approve`;

// List of inbound federated actions that took place.
export const FEDERATION_ACTIONS = `${API_LOCATION}federation/actions`;

export const API_STREAM_HEALTH_METRICS = `${API_LOCATION}metrics/video`;

// Save an array of stream keys
export const UPDATE_STREAM_KEYS = `${API_LOCATION}config/streamkeys`;

export const BOUNCECAST_STREAMERS = `${API_LOCATION}bouncecast/streamers`;

export const BOUNCECAST_STREAMER_UPDATE = `${API_LOCATION}bouncecast/streamers/update`;

export const BOUNCECAST_STREAMER_PASSWORD = `${API_LOCATION}bouncecast/streamers/password`;

export const BOUNCECAST_STREAM_KEYS = `${API_LOCATION}bouncecast/streamkeys`;

export const BOUNCECAST_STREAM_KEY_REVOKE = `${API_LOCATION}bouncecast/streamkeys/revoke`;

export const BOUNCECAST_LIVE_EVENTS = `${API_LOCATION}bouncecast/live-events`;

export const BOUNCECAST_NOTIFICATION_SUBSCRIBERS = `${API_LOCATION}bouncecast/notification-subscribers`;

export const BOUNCECAST_NOTIFICATION_SUBSCRIBER_DISABLE = `${API_LOCATION}bouncecast/notification-subscribers/disable`;

export const BOUNCECAST_NOTIFICATION_DELIVERIES = `${API_LOCATION}bouncecast/notification-deliveries`;

export const BOUNCECAST_NOTIFICATION_DELIVERIES_RETRY_FAILED = `${API_LOCATION}bouncecast/notification-deliveries/retry-failed`;

export const BOUNCECAST_NOTIFICATION_DELIVERIES_EXPORT = `${API_LOCATION}bouncecast/notification-deliveries/export`;

export const BOUNCECAST_EMAIL_SETTINGS = `${API_LOCATION}bouncecast/email-settings`;

export const BOUNCECAST_MESSENGER_SETTINGS = `${API_LOCATION}bouncecast/messenger-settings`;

export const BOUNCECAST_FACEBOOK_MESSENGER_ALERTS = `${API_LOCATION}bouncecast/integrations/facebook-messenger-alerts`;

export const BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_SETTINGS = `${API_LOCATION}bouncecast/integrations/facebook-messenger-alerts/settings`;

export const BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_VALIDATE = `${API_LOCATION}bouncecast/integrations/facebook-messenger-alerts/validate`;

export const BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_TEST = `${API_LOCATION}bouncecast/integrations/facebook-messenger-alerts/test-message`;

export const BOUNCECAST_FACEBOOK_MESSENGER_ALERTS_PREVIEW = `${API_LOCATION}bouncecast/integrations/facebook-messenger-alerts/preview`;

export const BOUNCECAST_PUSH_SETTINGS = `${API_LOCATION}bouncecast/push-settings`;

export const BOUNCECAST_MOBILE_ADMIN = `${API_LOCATION}bouncecast/mobile`;

export const BOUNCECAST_MOBILE_SETTINGS = `${API_LOCATION}bouncecast/mobile/settings`;

export const BOUNCECAST_ADMIN_SESSION = `${API_LOCATION}bouncecast/session`;

export const BOUNCECAST_SCHEDULE = `${API_LOCATION}bouncecast/schedule`;

export const BOUNCECAST_SCHEDULE_REMINDERS = `${API_LOCATION}bouncecast/schedule/reminders`;

export const BOUNCECAST_SCHEDULE_REMINDER_DISABLE = `${API_LOCATION}bouncecast/schedule/reminders/disable`;

export const BOUNCECAST_AUDIT_EVENTS = `${API_LOCATION}bouncecast/audit-events`;

export const BOUNCECAST_STARS_ADMIN = `${API_LOCATION}bouncecast/stars`;

export const BOUNCECAST_STARS_SETTINGS = `${API_LOCATION}bouncecast/stars/settings`;

export const BOUNCECAST_STARS_PACKAGES = `${API_LOCATION}bouncecast/stars/packages`;

export const BOUNCECAST_STARS_WALLET_ADJUST = `${API_LOCATION}bouncecast/stars/wallets/adjust`;

export const BOUNCECAST_STARS_TEST_OVERLAY = `${API_LOCATION}bouncecast/stars/test-overlay`;

export const BOUNCECAST_REWARDS_ADMIN = `${API_LOCATION}bouncecast/rewards`;

export const BOUNCECAST_REWARDS_SETTINGS = `${API_LOCATION}bouncecast/rewards/settings`;

export const BOUNCECAST_REWARDS_PRIZES = `${API_LOCATION}bouncecast/rewards/prizes`;

export const BOUNCECAST_REWARDS_CREDITS_ADJUST = `${API_LOCATION}bouncecast/rewards/credits/adjust`;

export const BOUNCECAST_REWARDS_TOP_SUPPORTERS_AWARD = `${API_LOCATION}bouncecast/rewards/top-supporters/award`;

export const BOUNCECAST_REWARDS_ORDER_UPDATE = `${API_LOCATION}bouncecast/rewards/orders/update`;

export const BOUNCECAST_REWARDS_ORDER_DISPATCH = `${API_LOCATION}bouncecast/rewards/orders/dispatch`;

export const BOUNCECAST_REWARDS_ORDERS_EXPORT = `${API_LOCATION}bouncecast/rewards/orders/export`;

export const BOUNCECAST_REWARDS_MESSAGE_READ = `${API_LOCATION}bouncecast/rewards/messages/read`;

export const BOUNCECAST_REWARDS_TASKS = `${API_LOCATION}bouncecast/rewards/tasks`;

export const BOUNCECAST_REWARDS_ACHIEVEMENTS = `${API_LOCATION}bouncecast/rewards/achievements`;

export const BOUNCECAST_USERS = `${API_LOCATION}bouncecast/users`;

export const BOUNCECAST_USER_PERMISSIONS = `${API_LOCATION}bouncecast/users/permissions`;

export const BOUNCECAST_COMMAND_CENTER = `${API_LOCATION}bouncecast/command-center`;

export const STARS_CONFIG = `${NEXT_PUBLIC_API_HOST}api/stars/config`;

export const STARS_LEADERBOARD = `${NEXT_PUBLIC_API_HOST}api/stars/leaderboard`;

export const STARS_WALLET = `${NEXT_PUBLIC_API_HOST}api/stars/wallet`;

export const STARS_PAYPAL_ORDER = `${NEXT_PUBLIC_API_HOST}api/stars/paypal/order`;

export const STARS_PAYPAL_CAPTURE = `${NEXT_PUBLIC_API_HOST}api/stars/paypal/capture`;

export const STARS_SEND = `${NEXT_PUBLIC_API_HOST}api/stars/send`;

export const REWARDS_WHEEL = `${NEXT_PUBLIC_API_HOST}api/rewards/wheel`;

export const REWARDS_BALANCE = `${NEXT_PUBLIC_API_HOST}api/rewards/balance`;

export const REWARDS_SPIN = `${NEXT_PUBLIC_API_HOST}api/rewards/spin`;

export const REWARDS_HISTORY = `${NEXT_PUBLIC_API_HOST}api/rewards/history`;

export const REWARDS_TASK_COMPLETE = `${NEXT_PUBLIC_API_HOST}api/rewards/tasks/complete`;

export const REWARDS_CLAIMS = `${NEXT_PUBLIC_API_HOST}api/rewards/claims`;

export const REWARDS_CLAIM_SUBMIT = `${NEXT_PUBLIC_API_HOST}api/rewards/claims/submit`;

export const REWARDS_NOTIFICATIONS = `${NEXT_PUBLIC_API_HOST}api/rewards/notifications`;

export const REWARDS_NOTIFICATION_READ = `${NEXT_PUBLIC_API_HOST}api/rewards/notifications/read`;

const STUDIO_API_LOCATION = `${NEXT_PUBLIC_API_HOST}api/bouncecast/studio/`;

export const BOUNCECAST_STUDIO_LOGIN = `${STUDIO_API_LOCATION}login`;

export const BOUNCECAST_STUDIO_REGISTER = `${STUDIO_API_LOCATION}register`;

export const BOUNCECAST_STUDIO_LOGOUT = `${STUDIO_API_LOCATION}logout`;

export const BOUNCECAST_STUDIO_ME = `${STUDIO_API_LOCATION}me`;

export const BOUNCECAST_STUDIO_SCHEDULE = `${STUDIO_API_LOCATION}schedule`;

export const BOUNCECAST_STUDIO_SCHEDULE_UPDATE = `${STUDIO_API_LOCATION}schedule/update`;

export const BOUNCECAST_STUDIO_SCHEDULE_CANCEL = `${STUDIO_API_LOCATION}schedule/cancel`;

export const BOUNCECAST_STUDIO_STREAM_KEYS = `${STUDIO_API_LOCATION}streamkeys`;

export const BOUNCECAST_STUDIO_STREAM_KEY_REVOKE = `${STUDIO_API_LOCATION}streamkeys/revoke`;

export const BOUNCECAST_STUDIO_LIVE_EVENTS = `${STUDIO_API_LOCATION}live-events`;

export const BOUNCECAST_STUDIO_PROFILE = `${STUDIO_API_LOCATION}profile`;

export const BOUNCECAST_ADMIN_LOGIN = `${NEXT_PUBLIC_API_HOST}api/bouncecast/admin/login`;

export const BOUNCECAST_AUTH_LOGIN = `${NEXT_PUBLIC_API_HOST}api/bouncecast/auth/login`;

export const BOUNCECAST_AUTH_LOGOUT = `${NEXT_PUBLIC_API_HOST}api/bouncecast/auth/logout`;

const ACCOUNT_API_LOCATION = `${NEXT_PUBLIC_API_HOST}api/bouncecast/account/`;

export const BOUNCECAST_ACCOUNT_REGISTER = `${ACCOUNT_API_LOCATION}register`;

export const BOUNCECAST_ACCOUNT_LOGIN = `${ACCOUNT_API_LOCATION}login`;

export const BOUNCECAST_ACCOUNT_ME = `${ACCOUNT_API_LOCATION}me`;

export const BOUNCECAST_ACCOUNT_PROFILE = `${ACCOUNT_API_LOCATION}profile`;

export const BOUNCECAST_ACCOUNT_PROFILE_IMAGE = `${ACCOUNT_API_LOCATION}profile-image`;

export const BOUNCECAST_ACCOUNT_NOTIFICATIONS = `${ACCOUNT_API_LOCATION}notifications`;

export const BOUNCECAST_ACCOUNT_HUB = `${ACCOUNT_API_LOCATION}hub`;

export const BOUNCECAST_PUBLIC_DJS = `${NEXT_PUBLIC_API_HOST}api/bouncecast/djs`;

export const BOUNCECAST_PUBLIC_SCHEDULE = `${NEXT_PUBLIC_API_HOST}api/bouncecast/schedule`;

export const BOUNCECAST_PUBLIC_SCHEDULE_REMINDERS = `${NEXT_PUBLIC_API_HOST}api/bouncecast/schedule/reminders`;

export const BOUNCECAST_MESSENGER_ALERTS_PUBLIC_CONFIG = `${NEXT_PUBLIC_API_HOST}api/bouncecast/messenger-alerts/config`;

export const API_YP_RESET = `${API_LOCATION}yp/reset`;

const GITHUB_RELEASE_URL = 'https://api.github.com/repos/owncast/owncast/releases/latest';

interface FetchOptions {
  data?: unknown;
  method?: string;
  auth?: boolean;
}

function getCredentialSafeFetchURL(url: string): string {
  if (typeof window === 'undefined') {
    return url;
  }

  const safeBaseURL = `${window.location.protocol}//${window.location.host}`;
  const requestURL = new URL(url, safeBaseURL);
  requestURL.username = '';
  requestURL.password = '';

  return requestURL.toString();
}

export async function fetchData(url: string, options?: FetchOptions) {
  const { data, method = 'GET', auth = true } = options || {};

  // eslint-disable-next-line no-undef
  const requestOptions: RequestInit = {
    method,
  };
  const headers: Record<string, string> = {
    Accept: 'application/json',
  };

  if (data !== undefined) {
    requestOptions.body = JSON.stringify(data);
    headers['Content-Type'] = 'application/json';
  }

  if (auth && ADMIN_USERNAME && ADMIN_STREAMKEY) {
    const encoded = btoa(`${ADMIN_USERNAME}:${ADMIN_STREAMKEY}`);
    headers.Authorization = `Basic ${encoded}`;
    requestOptions.mode = 'cors';
    requestOptions.credentials = 'include';
  }
  requestOptions.headers = headers;

  const response = await fetch(getCredentialSafeFetchURL(url), requestOptions);
  const json = await response.json();

  if (!response.ok) {
    const message = json.message || `An error has occurred: ${response.status}`;
    throw new Error(message);
  }
  return json;
}

export async function getUnauthedData(url: string, options?: FetchOptions) {
  const opts = {
    method: 'GET',
    auth: false,
    ...options,
  };
  return fetchData(url, opts);
}

export async function postUnauthedFormData(url: string, data: FormData) {
  const response = await fetch(getCredentialSafeFetchURL(url), {
    method: 'POST',
    headers: {
      Accept: 'application/json',
    },
    body: data,
  });
  const json = await response.json();
  if (!response.ok) {
    const message = json.message || json.error || `An error has occurred: ${response.status}`;
    throw new Error(message);
  }
  return json;
}

export async function fetchStudioData(url: string, token?: string, options?: FetchOptions) {
  const { data, method = 'GET' } = options || {};
  const headers: Record<string, string> = {
    Accept: 'application/json',
  };
  // eslint-disable-next-line no-undef
  const requestOptions: RequestInit = {
    method,
    headers,
  };

  if (data !== undefined) {
    requestOptions.body = JSON.stringify(data);
    headers['Content-Type'] = 'application/json';
  }

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(getCredentialSafeFetchURL(url), requestOptions);
  const json = await response.json();
  if (!response.ok) {
    const message = json.message || json.error || `An error has occurred: ${response.status}`;
    throw new Error(message);
  }
  return json;
}

export async function fetchExternalData(url: string) {
  try {
    const response = await fetch(url, {
      referrerPolicy: 'no-referrer', // Send no referrer header for privacy reasons.
      referrer: '',
    });
    if (!response.ok) {
      const message = `An error has occured: ${response.status}`;
      throw new Error(message);
    }
    const json = await response.json();
    return json;
  } catch (error) {
    console.log(error);
  }
  return {};
}

export async function getGithubRelease() {
  return fetchExternalData(GITHUB_RELEASE_URL);
}

function upToDate(local, remote) {
  return !semverGt(remote, local);
}

// Make a request to the server status API and the Github releases API
// and return a release if it's newer than the server version.
export async function upgradeVersionAvailable(currentVersion) {
  const recentRelease = await getGithubRelease();
  let recentReleaseVersion = recentRelease.tag_name;

  if (recentReleaseVersion.substr(0, 1) === 'v') {
    recentReleaseVersion = recentReleaseVersion.substr(1);
  }

  if (!upToDate(currentVersion, recentReleaseVersion)) {
    return recentReleaseVersion;
  }

  return null;
}
