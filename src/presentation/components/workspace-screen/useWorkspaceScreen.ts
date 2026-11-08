'use client';

import {useState} from 'react';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

export function useWorkspaceScreen(): {
  handleEncrypted: (message: MessageDto) => void;
} {
  const [, setMessageJournal] = useState<readonly MessageDto[]>([]);

  function handleEncrypted(message: MessageDto): void {
    setMessageJournal((journal) => [...journal, message]);
  }

  return {handleEncrypted};
}
