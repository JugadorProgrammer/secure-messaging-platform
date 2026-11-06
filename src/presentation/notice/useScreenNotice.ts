'use client';

import {useCallback, useState} from 'react';

import {NoticeHolder, ScreenNotice} from './ScreenNotice';

export function useScreenNotice(): {
  notice: ScreenNotice | null;
  show: (message: string) => void;
  dismiss: () => void;
} {
  const [holder] = useState(() => new NoticeHolder());
  const [notice, setNotice] = useState<ScreenNotice | null>(null);

  const show = useCallback((message: string) => {
    holder.show(message);
    setNotice(holder.notice);
  }, [holder]);

  const dismiss = useCallback(() => {
    holder.clear();
    setNotice(holder.notice);
  }, [holder]);

  return {notice, show, dismiss};
}
