'use client';

import {useEffect, useState} from 'react';
import clsx from 'clsx';

import styles from './Notification.module.css';

type NotificationProps = {
  message: string;
};

const VISIBLE_MS = 10000;
const LEAVE_MS = 500;

export function Notification({message}: NotificationProps) {
  const [isLeaving, setIsLeaving] = useState(false);
  const [isGone, setIsGone] = useState(false);

  useEffect(() => {
    const leaveTimer = setTimeout(() => {
      setIsLeaving(true);
    }, VISIBLE_MS);
    const hideTimer = setTimeout(() => {
      setIsGone(true);
    }, VISIBLE_MS + LEAVE_MS);

    return () => {
      clearTimeout(leaveTimer);
      clearTimeout(hideTimer);
    };
  }, []);

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
