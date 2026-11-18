'use client';

import clsx from 'clsx';

import {useNotification} from './useNotification';

import styles from './Notification.module.css';

type NotificationProps = {
  message: string;
};

export function Notification({message}: NotificationProps) {
  const {isLeaving, isGone} = useNotification();

  if (isGone) {
    return null;
  }

  return (
    <p
      className={clsx(styles['notification'], isLeaving && styles['notification--leaving'])}
      role="status"
    >
      {message}
    </p>
  );
}
