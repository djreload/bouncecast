/* eslint-disable import/prefer-default-export */
export class User {
  constructor(u) {
    this.id = u.id;
    this.displayName = u.displayName;
    this.email = u.email;
    this.profileImageUrl = u.profileImageUrl;
    this.displayColor = u.displayColor;
    this.createdAt = u.createdAt;
    this.previousNames = u.previousNames;
    this.nameChangedAt = u.nameChangedAt;
    this.scopes = u.scopes;
    this.authenticated = u.authenticated;
    this.isBot = u.isBot;
    this.registeredAt = u.registeredAt;
    this.lastLoginAt = u.lastLoginAt;

    if (this.scopes && this.scopes.length > 0) {
      this.isModerator = this.scopes.includes('MODERATOR');
      this.isOwner = this.scopes.includes('OWNER');
      this.isAdmin = this.scopes.includes('ADMIN');
      this.isDJ = this.scopes.includes('DJ');
    }
  }

  id: string;

  displayName: string;

  email?: string;

  profileImageUrl?: string;

  displayColor: number;

  createdAt: Date;

  previousNames: string[];

  nameChangedAt: Date;

  scopes: string[];

  authenticated: boolean;

  isBot: boolean;

  isModerator: boolean;

  isOwner?: boolean;

  isAdmin?: boolean;

  isDJ?: boolean;

  registeredAt?: Date;

  lastLoginAt?: Date;
}
