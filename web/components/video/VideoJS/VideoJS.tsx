import React, { FC } from 'react';
import videojs from 'video.js';
import type VideoJsPlayer from 'video.js/dist/types/player';
import { useTranslation } from 'next-export-i18n';

import styles from './VideoJS.module.scss';

require('video.js/dist/video-js.css');

const AUDIO_SESSION_CHANNEL = 'bouncecast-player-audio-session';

type AudioSessionMessage = {
  id?: string;
  type?: 'playing' | 'pause';
};

type BounceCastPlayerWindow = Window & {
  bouncecastVideoPlayers?: Map<string, VideoJsPlayer>;
};

const getActivePlayers = () => {
  const bouncecastWindow = window as BounceCastPlayerWindow;
  if (!bouncecastWindow.bouncecastVideoPlayers) {
    bouncecastWindow.bouncecastVideoPlayers = new Map<string, VideoJsPlayer>();
  }
  return bouncecastWindow.bouncecastVideoPlayers;
};

const SHORTCUT_SUFFIXES: Record<string, string> = {
  Play: ' (Space)',
  Pause: ' (Space)',
  Mute: ' (m)',
  Unmute: ' (m)',
  Fullscreen: ' (f)',
  'Non-Fullscreen': ' (f)',
  'Picture-in-Picture': ' (i)',
  'Exit Picture-in-Picture': ' (i)',
};

export type VideoJSProps = {
  options: any;
  onReady: (player: VideoJsPlayer, vjsInstance: typeof videojs) => void;
};

export const VideoJS: FC<VideoJSProps> = ({ options, onReady }) => {
  const videoRef = React.useRef<HTMLVideoElement | null>(null);
  const playerRef = React.useRef<VideoJsPlayer | null>(null);
  const playerIdRef = React.useRef(
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2),
  );
  const channelRef = React.useRef<BroadcastChannel | null>(null);
  const remotePauseRef = React.useRef(false);
  const { t } = useTranslation();

  const addShortcutsToLanguage = (vjs: typeof videojs, langCode: string) => {
    const updates: Record<string, string> = {};
    Object.keys(SHORTCUT_SUFFIXES).forEach(key => {
      const currentLabel = key;
      const suffix = SHORTCUT_SUFFIXES[key];
      updates[key] = t(`${currentLabel}${suffix}`);
    });
    vjs.addLanguage(langCode, updates);
  };

  React.useEffect(() => {
    const playerId = playerIdRef.current;

    const pausePlayer = (player: VideoJsPlayer | null) => {
      if (!player || player.isDisposed() || player.paused()) {
        return;
      }

      remotePauseRef.current = true;
      player.pause();
      window.setTimeout(() => {
        remotePauseRef.current = false;
      }, 0);
    };

    const broadcastAudioSession = (type: 'playing' | 'pause') => {
      if (remotePauseRef.current) {
        return;
      }

      channelRef.current?.postMessage({ id: playerId, type });
    };

    if ('BroadcastChannel' in window) {
      channelRef.current = new BroadcastChannel(AUDIO_SESSION_CHANNEL);
      channelRef.current.onmessage = event => {
        const data = event?.data as AudioSessionMessage;
        if (!data || data.id === playerId) {
          return;
        }

        if (data.type === 'playing' || data.type === 'pause') {
          pausePlayer(playerRef.current);
        }
      };
    }

    // Make sure Video.js player is only initialized once
    if (!playerRef.current) {
      const videoElement = videoRef.current;

      addShortcutsToLanguage(videojs, 'en');
      const finalOptions = {
        ...options,
        noUITitleAttributes: true, // Prevents videojs from adding a title attribute to UI elements, thus preventing "double tooltips".
      };
      // eslint-disable-next-line no-multi-assign
      const player: VideoJsPlayer = (playerRef.current = videojs(
        videoElement,
        finalOptions,
        () => onReady && onReady(player, videojs),
      ));

      const activePlayers = getActivePlayers();
      activePlayers.forEach((activePlayer, activePlayerId) => {
        if (activePlayerId !== playerId) {
          pausePlayer(activePlayer);
        }
      });
      activePlayers.set(playerId, player);

      player.on('playing', () => broadcastAudioSession('playing'));
      player.on('pause', () => broadcastAudioSession('pause'));

      player.autoplay(options.autoplay);
      player.src(options.sources);
    }

    return () => {
      channelRef.current?.close();
      channelRef.current = null;

      const activePlayers = (window as BounceCastPlayerWindow).bouncecastVideoPlayers;
      if (activePlayers instanceof Map) {
        activePlayers.delete(playerId);
      }

      if (playerRef.current && !playerRef.current.isDisposed()) {
        playerRef.current.dispose();
        playerRef.current = null;
      }
    };
    // Video.js owns its internal state after setup. Reinitializing on each render can create
    // overlapping playback instances, so this effect intentionally runs only for this mount.
  }, []);

  React.useEffect(() => {
    videojs.getPlayer(videoRef.current).on('xhr-hooks-ready', () => {
      const cachebusterRequestHook = o => {
        const { uri } = o;
        let updatedURI = uri;
        if (o.uri.match('m3u8')) {
          const u = uri.startsWith('http')
            ? new URL(uri)
            : new URL(uri, window.location.protocol + window.location.host);
          const cachebuster = Math.random().toString(16).slice(2, 8);
          u.searchParams.append('cachebust', cachebuster);
          updatedURI = u.toString();
        }
        return {
          ...o,
          uri: updatedURI,
        };
      };
      (
        videojs.getPlayer(videoRef.current).tech({ IWillNotUseThisInPlugins: true }) as any
      )?.vhs.xhr.onRequest(cachebusterRequestHook);
    });
  }, []);

  return (
    <div data-vjs-player>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        ref={videoRef}
        className={`video-js vjs-big-play-centered vjs-show-big-play-button-on-pause ${styles.player} vjs-owncast`}
      />
    </div>
  );
};
