import { FC, useEffect, useState } from 'react';
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

  useEffect(() => {
    if (!events.length) {
      return undefined;
    }
    const nextEvent = events[events.length - 1];
    setActiveEvent(nextEvent);
    if (nextEvent.soundEnabled) {
      playStarSound();
    }
    const timer = window.setTimeout(() => setActiveEvent(null), 4200);
    return () => window.clearTimeout(timer);
  }, [events]);

  if (!activeEvent) {
    return null;
  }

  return (
    <div className={styles.root} aria-live="polite">
      <div className={styles.toast}>
        <span className={styles.amount}>{activeEvent.amount} Stars</span>
        <span className={styles.sender}>{activeEvent.displayName}</span>
        {activeEvent.message && <p className={styles.message}>{activeEvent.message}</p>}
      </div>
    </div>
  );
};
