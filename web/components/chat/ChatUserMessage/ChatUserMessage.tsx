import { FC, ReactNode, useState } from 'react';
import cn from 'classnames';
import { Tooltip } from 'antd';
import { useRecoilValue } from 'recoil';
import dynamic from 'next/dynamic';
import { Interweave } from 'interweave';
import { UrlMatcher } from 'interweave-autolink';
import { ChatMessageHighlightMatcher } from './customMatcher';
import { ChatMessageEmojiMatcher } from './emojiMatcher';
import { ChatMessageTenorGifMatcher, renderTenorGifEmbeds } from './tenorMatcher';
import styles from './ChatUserMessage.module.scss';
import { formatTimestamp } from './messageFmt';
import { ChatMessage } from '../../../interfaces/chat-message.model';
import { accessTokenAtom, websocketServiceAtom } from '../../stores/ClientConfigStore';
import { User } from '../../../interfaces/user.model';
import { AuthedUserBadge } from '../ChatUserBadge/AuthedUserBadge';
import { ModerationBadge } from '../ChatUserBadge/ModerationBadge';
import { BotUserBadge } from '../ChatUserBadge/BotUserBadge';
import { MessageType } from '../../../interfaces/socket-events';
import WebsocketService from '../../../services/websocket-service';

// Lazy loaded components

const ChatModerationActionMenu = dynamic(
  () =>
    import('../ChatModerationActionMenu/ChatModerationActionMenu').then(
      mod => mod.ChatModerationActionMenu,
    ),
  {
    ssr: false,
  },
);

const reactionOptions = [
  { emoji: '\u{1F525}', label: 'Fire' },
  { emoji: '\u{2764}\u{FE0F}', label: 'Heart' },
  { emoji: '\u{1F602}', label: 'Laugh' },
  { emoji: '\u{1F44D}', label: 'Thumbs up' },
  { emoji: '\u{1F622}', label: 'Cry' },
];

export type ChatUserMessageProps = {
  message: ChatMessage;
  showModeratorMenu: boolean;
  highlightString: string;
  sentBySelf: boolean;
  sameUserAsLast: boolean;
  isAuthorModerator: boolean;
  isAuthorAuthenticated: boolean;
  isAuthorBot: boolean;
};

export type UserTooltipProps = {
  user: User;
  children: ReactNode;
};

const UserTooltip: FC<UserTooltipProps> = ({ children, user }) => {
  const { displayName, createdAt } = user;
  const content = `${displayName} first joined ${formatTimestamp(createdAt)}`;

  return (
    <Tooltip title={content} placement="topLeft" mouseEnterDelay={1}>
      {children}
    </Tooltip>
  );
};

export const ChatUserMessage: FC<ChatUserMessageProps> = ({
  message,
  highlightString,
  showModeratorMenu,
  sentBySelf, // Move the border to the right and render a background
  sameUserAsLast,
  isAuthorModerator,
  isAuthorAuthenticated,
  isAuthorBot,
}) => {
  const { id: messageId, body, user, timestamp } = message;
  const { id: userId, displayName, displayColor, profileImageUrl } = user;
  const accessToken = useRecoilValue<string>(accessTokenAtom);
  const websocketService = useRecoilValue<WebsocketService>(websocketServiceAtom);
  const [reactionPickerOpen, setReactionPickerOpen] = useState(false);
  const reactions = message.reactions || {};

  const color = `var(--theme-color-users-${displayColor})`;
  const formattedTimestamp = `Sent ${formatTimestamp(timestamp)}`;

  const badgeNodes = [];
  if (isAuthorModerator) {
    badgeNodes.push(<ModerationBadge key="mod" userColor={displayColor} />);
  }
  if (isAuthorAuthenticated) {
    badgeNodes.push(<AuthedUserBadge key="auth" userColor={displayColor} />);
  }
  if (isAuthorBot) {
    badgeNodes.push(<BotUserBadge key="bot" userColor={displayColor} />);
  }

  const addReaction = (emoji: string) => {
    if (websocketService?.isConnected()) {
      websocketService.send({
        type: MessageType.CHAT_REACTION,
        messageId,
        reaction: emoji,
      });
    }
    setReactionPickerOpen(false);
  };

  const handleMessageClick = (event: React.MouseEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement;
    if (target.closest('a, button')) {
      return;
    }

    setReactionPickerOpen(isOpen => !isOpen);
  };

  const handleMessageKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }

    event.preventDefault();
    setReactionPickerOpen(isOpen => !isOpen);
  };

  const activeReactions = Object.entries(reactions).filter(([, count]) => count > 0);

  return (
    <div
      className={cn(
        styles.messagePadding,
        sameUserAsLast && styles.messagePaddingCollapsed,
        'chat-message_user',
      )}
    >
      <div
        className={cn(styles.root, {
          [styles.ownMessage]: sentBySelf,
        })}
        style={{ borderColor: color }}
        title={formattedTimestamp}
        role="button"
        tabIndex={0}
        aria-label={`React to ${displayName}'s message`}
        onClick={handleMessageClick}
        onKeyDown={handleMessageKeyDown}
      >
        <div className={styles.background} style={{ color }} />

        <UserTooltip user={user}>
          <div className={sameUserAsLast ? styles.repeatUser : styles.user} style={{ color }}>
            {profileImageUrl && (
              <img src={profileImageUrl} alt="" loading="lazy" className={styles.profileImage} />
            )}
            <span className={styles.userName}>{displayName}</span>
            <span className={styles.userBadges}>{badgeNodes}</span>
          </div>
        </UserTooltip>
        <Tooltip mouseEnterDelay={1}>
          <Interweave
            className={styles.message}
            content={renderTenorGifEmbeds(body)}
            matchers={[
              new ChatMessageTenorGifMatcher('tenorGif'),
              new UrlMatcher('url', { customTLDs: ['online'] }),
              new ChatMessageHighlightMatcher('highlight', { highlightString }),
              new ChatMessageEmojiMatcher('emoji', { className: 'emoji' }),
            ]}
          />
        </Tooltip>
        {reactionPickerOpen && (
          <div className={styles.reactionTray} aria-label="Message reactions">
            {reactionOptions.map(reaction => (
              <button
                key={reaction.label}
                type="button"
                className={styles.reactionButton}
                aria-label={reaction.label}
                title={reaction.label}
                onClick={() => addReaction(reaction.emoji)}
              >
                {reaction.emoji}
              </button>
            ))}
          </div>
        )}
        {activeReactions.length > 0 && (
          <div className={styles.reactionSummary} aria-label="Selected reactions">
            {activeReactions.map(([emoji, count]) => (
              <span key={emoji} className={styles.reactionPill}>
                {emoji} {count}
              </span>
            ))}
          </div>
        )}
        {showModeratorMenu && (
          <div className={styles.modMenuWrapper}>
            <ChatModerationActionMenu
              messageID={messageId}
              accessToken={accessToken}
              userID={userId}
              userDisplayName={displayName}
            />
          </div>
        )}
      </div>
    </div>
  );
};
