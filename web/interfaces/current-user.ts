export interface CurrentUser {
  id: string;
  displayName: string;
  email?: string;
  profileImageUrl?: string;
  displayColor: number;
  scopes?: string[];
  isModerator: boolean;
  isOwner?: boolean;
  isAdmin?: boolean;
  isDJ?: boolean;
}
