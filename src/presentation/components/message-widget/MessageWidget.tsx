'use client';

import type {Message} from '@/domain/message';

import {MessageField} from '@/presentation/components/message-widget/message-field/MessageField';
import {Notification} from '@/presentation/components/notification/Notification';

import {useMessageWidget} from './useMessageWidget';

import styles from './MessageWidget.module.css';

type MessageWidgetProps = {
  message: Message;
};

function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString('ru-RU');
}

export function MessageWidget({message}: MessageWidgetProps) {
  const {canCheckFromCiphertext, isChecking, notice, checkFromCiphertext} = useMessageWidget(message);

  return (
    <article className={styles['message-widget']}>
      <header className={styles['message-widget__header']}>
        <div className={styles['message-widget__heading']}>
          <h3 className={styles['message-widget__title']}>{message.method}</h3>
          <p className={styles['message-widget__timestamp']}>{formatTimestamp(message.timestamp)}</p>
        </div>
        <div className={styles['message-widget__actions']}>
          <button
            className={styles['message-widget__check']}
            type="button"
            onClick={checkFromCiphertext}
            disabled={!canCheckFromCiphertext || isChecking}
          >
            Проверить из шифртекста
          </button>
        </div>
      </header>
      <div className={styles['message-widget__content']}>
        <MessageField label="Исходный текст" value={message.plaintext} />
        <MessageField label="Шифртекст" value={message.ciphertext} />
        {message.keys ? (
          <div className={styles['message-widget__keys']}>
            {message.method === 'RSA-32768' ? (
              <>
                <MessageField label="Публичный ключ" value={message.keys.publicKey} />
                <MessageField label="Секретный ключ" value={message.keys.privateKey} />
              </>
            ) : (
              <>
                <MessageField label="Ключ (hex)" value={message.keys.key} />
                <MessageField label="IV (hex)" value={message.keys.iv} />
              </>
            )}
          </div>
        ) : null}
      </div>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </article>
  );
}
