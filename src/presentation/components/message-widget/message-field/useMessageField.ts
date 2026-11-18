'use client';

import {useEffect, useRef, useState} from 'react';

const COPY_FEEDBACK_MS = 3000;

export function useMessageField(value: string): {
  copied: boolean;
  copy: () => Promise<void>;
} {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  async function copy(): Promise<void> {
    await navigator.clipboard.writeText(value);

    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
    }

    setCopied(true);
    timeoutRef.current = setTimeout(() => {
      setCopied(false);
      timeoutRef.current = null;
    }, COPY_FEEDBACK_MS);
  }

  return {
    copied,
    copy,
  };
}
