import {
  BOUNCECAST_ACCOUNT_HUB,
  BOUNCECAST_PUBLIC_DJS,
  BOUNCECAST_PUBLIC_SCHEDULE,
  BOUNCECAST_PUBLIC_SCHEDULE_REMINDERS,
  getUnauthedData,
  fetchData,
} from '../utils/apis';
import { StarLeaderboardEntry, StarWalletSummary } from '../interfaces/stars.model';

function withAccessToken(url: string, accessToken?: string): string {
  if (!accessToken) {
    return url;
  }
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}accessToken=${encodeURIComponent(accessToken)}`;
}

export type BounceCastPublicDJ = {
  displayName: string;
  handle: string;
  avatarUrl?: string;
  bio?: string;
  genres: string[];
  socialLinks: { label: string; url: string }[];
  heroImageUrl?: string;
  seoTitle?: string;
  shareImageUrl?: string;
  featuredScheduleId?: number;
  featuredSchedule?: BounceCastScheduleItem;
  role: string;
  upcomingSet?: string;
  upcomingStarts?: string;
  upcomingCount: number;
  totalLiveEvents: number;
  lastLiveAt?: string;
};

export type BounceCastScheduleFilters = {
  limit?: number;
  handle?: string;
  status?: 'all' | 'planned' | 'live';
  q?: string;
  from?: string;
  to?: string;
  includePast?: boolean;
};

export type BounceCastScheduleItem = {
  id: number;
  title: string;
  description?: string;
  startsAt: string;
  endsAt?: string;
  timezone: string;
  status: string;
  streamer?: string;
  handle?: string;
  avatarUrl?: string;
};

export type BounceCastDJProfile = {
  dj: BounceCastPublicDJ;
  schedule: BounceCastScheduleItem[];
};

export type BounceCastDestination = {
  key: string;
  label: string;
  url: string;
  available: boolean;
  reason?: string;
};

export type BounceCastAccountHub = {
  user: {
    id: string;
    displayName: string;
    email?: string;
    profileImageUrl?: string;
    scopes?: string[];
  };
  permissions: string[];
  notificationPreferences: {
    email: boolean;
    browserPush: boolean;
    messenger: boolean;
    messengerDestination?: string;
  };
  destinations: BounceCastDestination[];
  stars?: StarWalletSummary;
  leaderboard: StarLeaderboardEntry[];
  djProfile?: BounceCastDJProfile;
  upcomingSchedule: BounceCastScheduleItem[];
  reminders: BounceCastReminderStatus[];
};

export type BounceCastReminderStatus = {
  id: number;
  scheduleId: number;
  title: string;
  streamer?: string;
  startsAt: string;
  email: boolean;
  browserPush: boolean;
  messenger: boolean;
  lastQueuedAt?: string;
  lastDeliveryStatus?: string;
  lastDeliveryError?: string;
  disabledAt?: string;
  createdAt: string;
  updatedAt: string;
};

export class BounceCastService {
  public static async getDJs(): Promise<BounceCastPublicDJ[]> {
    return getUnauthedData(BOUNCECAST_PUBLIC_DJS);
  }

  public static async getDJProfile(handle: string): Promise<BounceCastDJProfile> {
    return getUnauthedData(`${BOUNCECAST_PUBLIC_DJS}/${encodeURIComponent(handle)}`);
  }

  public static async getSchedule(
    limitOrFilters?: number | BounceCastScheduleFilters,
    handle?: string,
  ): Promise<BounceCastScheduleItem[]> {
    const requestedLimitOrFilters = limitOrFilters ?? 12;
    const filters =
      typeof requestedLimitOrFilters === 'number'
        ? ({ limit: requestedLimitOrFilters, handle } as BounceCastScheduleFilters)
        : requestedLimitOrFilters;
    const params = new URLSearchParams({ limit: String(filters.limit || 12) });
    if (filters.handle) {
      params.set('handle', filters.handle);
    }
    if (filters.status && filters.status !== 'all') {
      params.set('status', filters.status);
    }
    if (filters.q) {
      params.set('q', filters.q);
    }
    if (filters.from) {
      params.set('from', filters.from);
    }
    if (filters.to) {
      params.set('to', filters.to);
    }
    if (filters.includePast) {
      params.set('includePast', 'true');
    }
    return getUnauthedData(`${BOUNCECAST_PUBLIC_SCHEDULE}?${params.toString()}`);
  }

  public static async getAccountHub(accessToken: string): Promise<BounceCastAccountHub> {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_HUB, accessToken));
  }

  public static async setScheduleReminder(
    accessToken: string,
    scheduleId: number,
    channels: {
      email?: boolean;
      browserPush?: boolean;
      browserPushEndpoint?: string;
      messenger?: boolean;
    },
  ) {
    return fetchData(withAccessToken(BOUNCECAST_PUBLIC_SCHEDULE_REMINDERS, accessToken), {
      method: 'POST',
      auth: false,
      data: {
        scheduleId,
        email: Boolean(channels.email),
        browserPush: Boolean(channels.browserPush),
        browserPushEndpoint: channels.browserPushEndpoint || '',
        messenger: Boolean(channels.messenger),
      },
    });
  }
}

export default BounceCastService;
