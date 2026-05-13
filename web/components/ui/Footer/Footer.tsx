import { FC } from 'react';
import { useRecoilValue } from 'recoil';
import { useTranslation } from 'next-export-i18n';
import styles from './Footer.module.scss';
import { ServerStatus } from '../../../interfaces/server-status.model';
import { serverStatusState } from '../../stores/ClientConfigStore';
import { Localization } from '../../../types';
import Translation from '../Translation/Translation';

export const Footer: FC = () => {
  const clientStatus = useRecoilValue<ServerStatus>(serverStatusState);
  const { versionNumber } = clientStatus;
  const { t } = useTranslation();
  return (
    <footer className={styles.footer} id="footer">
      <span id="owncast-powered-by-text">
        <Translation
          translationKey={Localization.Common.poweredByOwncastVersion}
          vars={{ versionNumber }}
          defaultText="Powered by <a href='https://github.com/djreload/bouncecast'>BounceCast v{{versionNumber}}</a>, forked from <a href='https://owncast.online'>Owncast</a>"
        />
      </span>
      <span className={styles.links}>
        <a href="https://github.com/djreload/bouncecast" target="_blank" rel="noreferrer">
          {t(Localization.Frontend.Footer.documentation)}
        </a>
        <a href="https://github.com/djreload/bouncecast/issues" target="_blank" rel="noreferrer">
          {t(Localization.Frontend.Footer.contribute)}
        </a>
        <a href="https://github.com/djreload/bouncecast" target="_blank" rel="noreferrer">
          {t(Localization.Frontend.Footer.source)}
        </a>
      </span>
    </footer>
  );
};
