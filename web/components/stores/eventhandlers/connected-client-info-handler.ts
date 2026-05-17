import { ConnectedClientInfoEvent } from '../../../interfaces/socket-events';

export function handleConnectedClientInfoMessage(
  message: ConnectedClientInfoEvent,
  setChatAuthenticated: (boolean) => void,
  setCurrentUser: (CurrentUser) => void,
) {
  const { user } = message;
  const { id, displayName, email, profileImageUrl, displayColor, scopes, authenticated } = user;
  setChatAuthenticated(authenticated);

  setCurrentUser({
    id: id.toString(),
    displayName,
    email,
    profileImageUrl,
    displayColor,
    scopes,
    isModerator: scopes?.includes('MODERATOR'),
    isOwner: scopes?.includes('OWNER'),
    isAdmin: scopes?.includes('ADMIN'),
    isDJ: scopes?.includes('DJ'),
  });
}
