'use client';

import {useEffect, useState} from 'react';

const VISIBLE_MS = 10000;
const LEAVE_MS = 500;

export function useNotification(): {
  isLeaving: boolean;
  isGone: boolean;
} {
  const [isLeaving, setIsLeaving] = useState(false);
  const [isGone, setIsGone] = useState(false);

  useEffect(() => {
    const leaveTimer = window.setTimeout(() => {
      setIsLeaving(true);
    }, VISIBLE_MS);
    const hideTimer = window.setTimeout(() => {
      setIsGone(true);
    }, VISIBLE_MS + LEAVE_MS);

    return () => {
      window.clearTimeout(leaveTimer);
      window.clearTimeout(hideTimer);
    };
  }, []);

  return {
    isLeaving,
    isGone,
  };
}
