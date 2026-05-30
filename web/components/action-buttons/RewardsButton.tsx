import { Button } from 'antd';
import { FC } from 'react';
import dynamic from 'next/dynamic';
import { useTranslation } from 'next-export-i18n';
import styles from './ActionButton/ActionButton.module.scss';

const GiftFilled = dynamic(() => import('@ant-design/icons/GiftFilled'), {
  ssr: false,
});

export type RewardsButtonProps = {
  onClick?: () => void;
};

export const RewardsButton: FC<RewardsButtonProps> = ({ onClick }) => {
  const { t } = useTranslation();

  return (
    <Button
      type="primary"
      className={styles.button}
      icon={<GiftFilled />}
      onClick={onClick}
      id="rewards-wheel-button"
    >
      {t('Rewards Wheel')}
    </Button>
  );
};
