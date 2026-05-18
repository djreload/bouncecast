import classNames from 'classnames';
import React, { FC, useEffect, useRef, useState } from 'react';
import { useRecoilValue } from 'recoil';
import { starsOverlayEventsAtom } from '../stores/ClientConfigStore';
import { StarsSentSocketEvent } from '../../interfaces/socket-events';
import styles from './StarsOverlay.module.scss';

function playStarSound() {
  try {
    const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
    if (!AudioContextClass) {
      return;
    }
    const context = new AudioContextClass();
    const oscillator = context.createOscillator();
    const gain = context.createGain();
    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(660, context.currentTime);
    oscillator.frequency.exponentialRampToValueAtTime(990, context.currentTime + 0.18);
    gain.gain.setValueAtTime(0.0001, context.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.08, context.currentTime + 0.03);
    gain.gain.exponentialRampToValueAtTime(0.0001, context.currentTime + 0.35);
    oscillator.connect(gain);
    gain.connect(context.destination);
    oscillator.start();
    oscillator.stop(context.currentTime + 0.36);
  } catch {
    // Browser autoplay policies may reject audio without user interaction.
  }
}

export const StarsOverlay: FC = () => {
  const events = useRecoilValue<StarsSentSocketEvent[]>(starsOverlayEventsAtom);
  const [activeEvent, setActiveEvent] = useState<StarsSentSocketEvent>(null);
  const [queue, setQueue] = useState<StarsSentSocketEvent[]>([]);
  const seenEventIds = useRef<Set<string>>(new Set());

  useEffect(() => {
    const newEvents = events.filter(event => {
      if (!event.id || seenEventIds.current.has(event.id)) {
        return false;
      }
      seenEventIds.current.add(event.id);
      return true;
    });

    if (newEvents.length > 0) {
      setQueue(currentQueue => [...currentQueue, ...newEvents]);
    }
  }, [events]);

  useEffect(() => {
    if (activeEvent || queue.length === 0) {
      return undefined;
    }

    const nextEvent = queue[0];
    setActiveEvent(nextEvent);
    setQueue(currentQueue => currentQueue.slice(1));
    return undefined;
  }, [activeEvent, queue]);

  useEffect(() => {
    if (!activeEvent) {
      return undefined;
    }

    if (activeEvent.soundEnabled) {
      playStarSound();
    }
    const timer = window.setTimeout(() => {
      setActiveEvent(null);
    }, 4200);
    return () => window.clearTimeout(timer);
  }, [activeEvent]);

  if (!activeEvent) {
    return null;
  }

  return (
    <div className={styles.root} aria-live="polite">
      <div className={classNames(styles.toast, styles[activeEvent.effect] || styles.sparkle)}>
        <span className={styles.amount}>{activeEvent.amount} Stars &#11088;</span>
        <span className={styles.sender}>{activeEvent.displayName}</span>
        {activeEvent.message && <p className={styles.message}>{activeEvent.message}</p>}
      </div>
    </div>
  );
};
