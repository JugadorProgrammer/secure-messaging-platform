'use client';

import {useState} from 'react';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

import {MessageDecryptionWidget} from '@/presentation/components/message-decryption-widget/MessageDecryptionWidget';
import {MessageEncryptionWidget} from '@/presentation/components/message-encryption-widget/MessageEncryptionWidget';
import {MessageJournalWidget} from '@/presentation/components/message-journal-widget/MessageJournalWidget';

import styles from './WorkspaceScreen.module.css';

export function WorkspaceScreen() {
  const [, setMessageJournal] = useState<readonly MessageDto[]>([]);

  function handleEncrypted(message: MessageDto): void {
    setMessageJournal((journal) => [...journal, message]);
  }

  return (
    <main className={styles['workspace-screen']}>
      <h1 className={styles['workspace-screen__title']}>Рабочий стол</h1>
      <div className={styles['workspace-screen__crypto']}>
        <section className={styles['workspace-screen__widget']}>
          <MessageEncryptionWidget onEncrypted={handleEncrypted} />
        </section>
        <section className={styles['workspace-screen__widget']}>
          <MessageDecryptionWidget />
        </section>
      </div>
      <section className={styles['workspace-screen__widget']}>
        <MessageJournalWidget />
      </section>
    </main>
  );
}
