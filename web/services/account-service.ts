import {
  BOUNCECAST_ACCOUNT_ME,
  BOUNCECAST_ACCOUNT_NOTIFICATIONS,
  BOUNCECAST_ACCOUNT_PROFILE,
  BOUNCECAST_ACCOUNT_PROFILE_IMAGE,
  BOUNCECAST_ACCOUNT_REGISTER,
  BOUNCECAST_AUTH_LOGIN,
  getUnauthedData,
  postUnauthedFormData,
} from '../utils/apis';

function withAccessToken(url: string, accessToken?: string): string {
  if (!accessToken) {
    return url;
  }
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}accessToken=${encodeURIComponent(accessToken)}`;
}

export type AccountPayload = {
  displayName?: string;
  email?: string;
  password?: string;
  profileImageUrl?: string;
  notificationPreferences?: NotificationPreferencesPayload;
};

export type NotificationPreferencesPayload = {
  email?: boolean;
  browserPush?: boolean;
  browserPushEndpoint?: string;
  messenger?: boolean;
  messengerDestination?: string;
};

export class AccountService {
  public static async register(accessToken: string, payload: AccountPayload) {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_REGISTER, accessToken), {
      method: 'POST',
      data: payload,
    });
  }

  public static async login(payload: AccountPayload) {
    return getUnauthedData(BOUNCECAST_AUTH_LOGIN, {
      method: 'POST',
      data: payload,
    });
  }

  public static async me(accessToken: string) {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_ME, accessToken));
  }

  public static async updateProfile(accessToken: string, payload: AccountPayload) {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_PROFILE, accessToken), {
      method: 'POST',
      data: payload,
    });
  }

  public static async uploadProfileImage(accessToken: string, file: File) {
    const data = new FormData();
    data.append('image', file);
    return postUnauthedFormData(
      withAccessToken(BOUNCECAST_ACCOUNT_PROFILE_IMAGE, accessToken),
      data,
    );
  }

  public static async updateNotifications(
    accessToken: string,
    payload: NotificationPreferencesPayload,
  ) {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_NOTIFICATIONS, accessToken), {
      method: 'POST',
      data: payload,
    });
  }
}
