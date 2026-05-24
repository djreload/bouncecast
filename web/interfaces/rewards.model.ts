export interface RewardSettings {
  enabled: boolean;
  spinCost: number;
  chatRewardsEnabled: boolean;
  chatValidMessageCount: number;
  chatCooldownSeconds: number;
  chatCreditReward: number;
  topSupporterFirstCredits: number;
  topSupporterSecondCredits: number;
  topSupporterThirdCredits: number;
  overlayEnabled: boolean;
  overlayDurationSeconds: number;
  overlaySoundEnabled: boolean;
  overlayShowImage: boolean;
  overlayTemplate: string;
  debugLoggingEnabled?: boolean;
  futurePaidRewardsNote?: string;
  futureDiscountRewardsNote?: string;
  fulfilmentPrivacyStatement?: string;
}

export interface RewardPrize {
  id: number;
  name: string;
  description?: string;
  image?: string;
  prizeType: 'physical' | 'digital' | 'discount_future_placeholder' | 'sorry';
  oddsWeight: number;
  stockQuantity?: number;
  active: boolean;
  displayOrder: number;
  claimRequired: boolean;
  marketingConsentRequired: boolean;
  terms?: string;
  fulfilmentNotes?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface RewardSpinBalance {
  userId: string;
  displayName?: string;
  email?: string;
  profileImageUrl?: string;
  balance: number;
  lifetimeEarned: number;
  lifetimeSpent: number;
}

export interface RewardSpinLedgerEntry {
  id: number;
  userId: string;
  amount: number;
  balanceAfter: number;
  source: string;
  referenceId?: string;
  note?: string;
  createdAt: string;
}

export interface RewardSpin {
  id: number;
  userId: string;
  prizeId?: number;
  prizeSnapshot: string;
  imageSnapshot?: string;
  typeSnapshot: string;
  oddsSnapshot: number;
  ledgerId: number;
  resultType: string;
  createdAt: string;
}

export interface RewardClaim {
  id: number;
  userId: string;
  spinId: number;
  prizeId?: number;
  status: string;
  fullName?: string;
  addressLine1?: string;
  addressLine2?: string;
  townCity?: string;
  countyState?: string;
  postcode?: string;
  country?: string;
  email?: string;
  phone?: string;
  deliveryNotes?: string;
  marketingConsent: boolean;
  consentText?: string;
  prizeSnapshot?: string;
  imageSnapshot?: string;
  marketingConsentRequired?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface RewardOrder {
  id: number;
  winnerUserId: string;
  usernameSnapshot: string;
  prizeId?: number;
  prizeSnapshot: string;
  imageSnapshot?: string;
  spinId: number;
  claimId?: number;
  addressSnapshot?: string;
  orderStatus: string;
  dispatchStatus: string;
  adminNotes?: string;
  courier?: string;
  trackingReference?: string;
  trackingUrl?: string;
  dispatchNote?: string;
  dispatchedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RewardWinner {
  id: number;
  userId: string;
  usernameSnapshot: string;
  email?: string;
  prizeSnapshot: string;
  imageSnapshot?: string;
  typeSnapshot: string;
  spinId: number;
  claimId?: number;
  orderId?: number;
  notificationStatus: string;
  createdAt: string;
}

export interface RewardMessage {
  id: number;
  type: string;
  severity: string;
  title: string;
  body?: string;
  userId?: string;
  prizeId?: number;
  spinId?: number;
  claimId?: number;
  orderId?: number;
  readAt?: string;
  createdAt: string;
}

export interface RewardNotification {
  id: number;
  userId: string;
  type: string;
  orderId?: number;
  prizeId?: number;
  title: string;
  message?: string;
  emailSent: boolean;
  emailError?: string;
  readAt?: string;
  createdAt: string;
}

export interface RewardTask {
  id: number;
  title: string;
  description?: string;
  creditReward: number;
  active: boolean;
}

export interface RewardAchievement {
  id: number;
  name: string;
  conditionKey: string;
  rewardAmount: number;
  active: boolean;
}

export interface RewardWheelData {
  settings: RewardSettings;
  balance: RewardSpinBalance;
  prizes: RewardPrize[];
}

export interface RewardSpinResult {
  spin: RewardSpin;
  prize: RewardPrize;
  balance: number;
  winner?: RewardWinner;
  claim?: RewardClaim;
  order?: RewardOrder;
  overlaySent: boolean;
  message: string;
}

export interface RewardAdminSummary {
  settings: RewardSettings;
  prizes: RewardPrize[];
  balances: RewardSpinBalance[];
  ledger: RewardSpinLedgerEntry[];
  spins: RewardSpin[];
  winners: RewardWinner[];
  claims: RewardClaim[];
  orders: RewardOrder[];
  adminMessages: RewardMessage[];
  notifications: RewardNotification[];
  tasks: RewardTask[];
  achievements: RewardAchievement[];
  unreadCount: number;
}
