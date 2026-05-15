import React, { FC } from 'react';
import cn from 'classnames';
import styles from './OwncastLogo.module.scss';

export type LogoProps = {
  variant?: 'simple' | 'contrast';
  className?: string;
};

export const OwncastLogo: FC<LogoProps> = ({ variant = 'simple', className = '' }) => {
  const rootClassName = cn(styles.root, {
    [styles.simple]: variant === 'simple',
    [styles.contrast]: variant === 'contrast',
  });

  return (
    <div className={`${rootClassName} ${className}`}>
      <img src="/logo" alt="BounceCast" className={`${styles.logo} logo-svg`} />
    </div>
  );
};
