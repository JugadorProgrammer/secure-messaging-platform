'use client';

import {useEffect, useState} from 'react';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

import {getAllMessages} from '@/presentation/api/messages';

type MessageJournalLoadStatus = 'idle' | 'loading' | 'ok' | 'failed';

type MessageJournalWidgetState = {
  status: MessageJournalLoadStatus;
  messages: MessageDto[];
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

export function useMessageJournalWidget(): MessageJournalWidgetState {
  const [status, setStatus] = useState<MessageJournalLoadStatus>('loading');
  const [messages, setMessages] = useState<MessageDto[]>([]);

  useEffect(() => {
    let isCancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    function load(retry: number): void {
      void getAllMessages().then((result) => {
        if (isCancelled) {
          return;
        }

        if (result.isSuccess) {
          setMessages(result.success);
          setStatus('ok');
          return;
        }

        if (retry === RETRY_LIMIT) {
          setMessages([]);
          setStatus('failed');
          return;
        }

        timer = setTimeout(() => {
          load(retry + 1);
        }, RETRY_DELAY_MS);
      });
    }

    load(0);

    return () => {
      isCancelled = true;
      clearTimeout(timer);
    };
  }, []);

  return {
    status,
    messages,
  };
}
