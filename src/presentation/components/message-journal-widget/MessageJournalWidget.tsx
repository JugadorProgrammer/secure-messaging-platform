'use client';

import {Notification} from '@/presentation/components/notification/Notification';

import {useMessageJournalWidget} from './useMessageJournalWidget';

import styles from './MessageJournalWidget.module.css';

export function MessageJournalWidget() {
  const {status, messages} = useMessageJournalWidget();

  return (
    <div className={styles['message-journal-widget']}>
      <h2 className={styles['message-journal-widget__title']}>Журнал сообщений</h2>
      {status === 'loading' ? (
        <div className={styles['message-journal-widget__bars']} aria-busy="true">
          <span className={styles['message-journal-widget__bar']} />
          <span className={styles['message-journal-widget__bar']} />
          <span className={styles['message-journal-widget__bar']} />
        </div>
      ) : null}
      {status === 'failed' ? (
        <>
          <div className={styles['message-journal-widget__failed']} />
          <Notification message="Не удалось загрузить журнал сообщений." />
        </>
      ) : null}
      {status === 'ok' ? (
        <div className={styles['message-journal-widget__messages']}>
          {messages.length === 0 ? (
            <p className={styles['message-journal-widget__empty']}>Сообщений нет</p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
