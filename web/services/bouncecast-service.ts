import {
  BOUNCECAST_ACCOUNT_HUB,
  BOUNCECAST_PUBLIC_DJS,
  BOUNCECAST_PUBLIC_SCHEDULE,
  getUnauthedData,
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
  role: string;
  upcomingSet?: string;
  upcomingStarts?: string;
  upcomingCount: number;
  totalLiveEvents: number;
  lastLiveAt?: string;
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
};

export class BounceCastService {
  public static async getDJs(): Promise<BounceCastPublicDJ[]> {
    return getUnauthedData(BOUNCECAST_PUBLIC_DJS);
  }

  public static async getDJProfile(handle: string): Promise<BounceCastDJProfile> {
    return getUnauthedData(`${BOUNCECAST_PUBLIC_DJS}/${encodeURIComponent(handle)}`);
  }

  public static async getSchedule(limit = 12, handle?: string): Promise<BounceCastScheduleItem[]> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (handle) {
      params.set('handle', handle);
    }
    return getUnauthedData(`${BOUNCECAST_PUBLIC_SCHEDULE}?${params.toString()}`);
  }

  public static async getAccountHub(accessToken: string): Promise<BounceCastAccountHub> {
    return getUnauthedData(withAccessToken(BOUNCECAST_ACCOUNT_HUB, accessToken));
  }
}

export default BounceCastService;
