export interface StarPackage {
  id: number;
  name: string;
  starAmount: number;
  priceCents: number;
  currency: string;
  enabled: boolean;
  displayOrder: number;
}

export interface StarSettings {
  enabled: boolean;
  paypalEnvironment: 'sandbox' | 'live';
  paypalClientId: string;
  paypalClientSecret?: string;
  paypalWebhookId?: string;
  currency: string;
  supportMessage: string;
  minimumSendAmount: number;
  maximumSendAmount: number;
  sendCooldownSeconds: number;
  overlayEffectsEnabled: boolean;
  soundEffectsEnabled: boolean;
  debugLoggingEnabled?: boolean;
  packages?: StarPackage[];
}

export interface StarWallet {
  userId: string;
  balance: number;
  lifetimePurchased: number;
  lifetimeSent: number;
}

export interface StarWalletTransaction {
  id: number;
  transactionType: string;
  amount: number;
  balanceAfter: number;
  referenceType?: string;
  referenceId?: string;
  notes?: string;
  createdAt: string;
}

export interface StarWalletSummary {
  wallet: StarWallet;
  transactions: StarWalletTransaction[];
}

export interface StarSendEvent {
  id: number;
  displayName: string;
  amount: number;
  message?: string;
  effect: string;
  createdAt: string;
}

export interface StarLeaderboardEntry {
  rank: number;
  userId: string;
  displayName: string;
  totalSent: number;
  sendCount: number;
  lastSentAt: string;
}

export interface StarAdminSummary {
  settings: StarSettings;
  packages: StarPackage[];
  leaderboard: StarLeaderboardEntry[];
  orders: any[];
  sendEvents: StarSendEvent[];
  transactions: StarWalletTransaction[];
  wallets: StarWallet[];
}
