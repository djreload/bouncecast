import classNames from 'classnames';
import React, { FC, useEffect, useRef, useState } from 'react';
import { useRecoilValue } from 'recoil';
import { starsOverlayEventsAtom } from '../stores/ClientConfigStore';
import { StarsSentSocketEvent } from '../../interfaces/socket-events';
import styles from './StarsOverlay.module.scss';

const particleCountByEffect: Record<string, number> = {
  sparkle: 18,
  fireworks: 24,
  hearts: 16,
  hype: 12,
  dj_drop: 18,
};

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

function renderEffectLayer(effect: string) {
  const count = particleCountByEffect[effect] || particleCountByEffect.sparkle;

  return (
    <div className={styles.effectLayer} aria-hidden="true">
      {Array.from({ length: count }, (_, index) => {
        const angle = (Math.PI * 2 * index) / count;
        const radius = 68 + (index % 5) * 18;
        const style = {
          '--i': index,
          '--x': `${Math.cos(angle) * radius}px`,
          '--y': `${Math.sin(angle) * radius}px`,
          '--x-far': `${Math.cos(angle) * radius * 1.22}px`,
          '--y-far': `${Math.sin(angle) * radius * 1.22}px`,
          '--delay': `${(index % 8) * 0.08}s`,
          '--size': `${8 + (index % 4) * 3}px`,
          '--heart-size': `${12 + (index % 4) * 4}px`,
          '--record-size': `${30 + (index % 4) * 6}px`,
          '--height': `${0.72 + (index % 5) * 0.08}`,
          '--bar-left': `${-130 + index * 24}px`,
        } as React.CSSProperties;

        return (
          <span
            key={`${effect}-${index}`}
            className={styles.effectParticle}
            data-effect-particle={effect}
            style={style}
          />
        );
      })}
    </div>
  );
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

  const effectClass = styles[activeEvent.effect] || styles.sparkle;

  return (
    <div className={classNames(styles.root, effectClass)} aria-live="polite">
      {renderEffectLayer(activeEvent.effect)}
      <div
        className={classNames(styles.toast, effectClass)}
        data-effect={activeEvent.effect}
        data-testid="stars-overlay-toast"
      >
        <span className={styles.amount}>{activeEvent.amount} Stars &#11088;</span>
        <span className={styles.sender}>{activeEvent.displayName}</span>
        {activeEvent.message && <p className={styles.message}>{activeEvent.message}</p>}
      </div>
    </div>
  );
};
