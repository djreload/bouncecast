import React, { FC, useEffect, useState } from 'react';
import { BOUNCECAST_MESSENGER_ALERTS_PUBLIC_CONFIG, getUnauthedData } from '../../utils/apis';
import styles from './MessengerAlertsCTA.module.scss';

type MessengerAlertsConfig = {
  enabled: boolean;
  pageUrl?: string;
  optInKeyword: string;
  message: string;
};

export const MessengerAlertsCTA: FC = () => {
  const [config, setConfig] = useState<MessengerAlertsConfig | null>(null);

  useEffect(() => {
    getUnauthedData(BOUNCECAST_MESSENGER_ALERTS_PUBLIC_CONFIG)
      .then(result => setConfig(result))
      .catch(() => setConfig(null));
  }, []);

  if (!config?.enabled || !config.pageUrl) {
    return null;
  }

  return (
    <aside className={styles.messengerCta}>
      <div className={styles.copy}>
        <p className={styles.title}>Get Messenger live alerts</p>
        <p className={styles.body}>
          {config.message ||
            'Get a Messenger notification when we go live. You can opt out anytime.'}{' '}
          Message {config.optInKeyword || 'LIVE'} to subscribe.
        </p>
      </div>
      <a className={styles.button} href={config.pageUrl} target="_blank" rel="noreferrer">
        Message LIVE
      </a>
    </aside>
  );
};
