import Link from 'next/link';
import { useEffect, useState } from 'react';
import { BounceCastScheduleItem, BounceCastService } from '../../services/bouncecast-service';
import styles from './PublicLineup.module.scss';

function formatLineupTime(value: string) {
  return new Date(value).toLocaleString([], {
    weekday: 'short',
    hour: '2-digit',
    minute: '2-digit',
    day: 'numeric',
    month: 'short',
  });
}

export function PublicLineup() {
  const [schedule, setSchedule] = useState<BounceCastScheduleItem[]>([]);

  useEffect(() => {
    BounceCastService.getSchedule(3)
      .then(result => setSchedule(result || []))
      .catch(() => setSchedule([]));
  }, []);

  return (
    <section className={styles.lineup} aria-label="Upcoming DJ lineup">
      <div className={styles.header}>
        <div>
          <p className={styles.eyebrow}>Upcoming sets</p>
          <h2 className={styles.title}>Lineup</h2>
        </div>
        <Link href="/djs" className={styles.link}>
          View DJ profiles
        </Link>
      </div>

      {schedule.length > 0 ? (
        <div className={styles.grid}>
          {schedule.map(item => (
            <article className={styles.card} key={item.id}>
              <p className={styles.time}>{formatLineupTime(item.startsAt)}</p>
              <p className={styles.setTitle}>{item.title}</p>
              <p className={styles.meta}>
                {item.streamer ? `with ${item.streamer}` : 'BounceCast DJ set'}
              </p>
            </article>
          ))}
        </div>
      ) : (
        <p className={styles.empty}>No public sets are scheduled yet.</p>
      )}
    </section>
  );
}

export default PublicLineup;
