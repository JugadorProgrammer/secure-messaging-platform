'use client';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

import {MessageField} from '@/presentation/components/message-widget/message-field/MessageField';
import {Notification} from '@/presentation/components/notification/Notification';

import {useMessageWidget} from './useMessageWidget';

import styles from './MessageWidget.module.css';

type MessageWidgetProps = {
  message: MessageDto;
};

function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString('ru-RU');
}

export function MessageWidget({message}: MessageWidgetProps) {
  const {isChecking, isExportingPdf, notice, checkFromCiphertext, exportPdf} =
    useMessageWidget(message);

  return (
    <article className={styles['message-widget']}>
      <header className={styles['message-widget__header']}>
        <div className={styles['message-widget__heading']}>
          <h3 className={styles['message-widget__title']}>{message.method}</h3>
          <p className={styles['message-widget__meta']}>
            {message.user.name} · {formatTimestamp(message.timestamp)}
          </p>
        </div>
        <div className={styles['message-widget__actions']}>
          <button
            className={styles['message-widget__action']}
            type="button"
            onClick={checkFromCiphertext}
            disabled={isChecking}
          >
            Проверить из шифртекста
          </button>
          <button
            className={styles['message-widget__action']}
            type="button"
            onClick={exportPdf}
            disabled={isExportingPdf}
          >
            PDF + ЭЦП
          </button>
        </div>
      </header>
      <div className={styles['message-widget__content']}>
        <MessageField label="Исходный текст" value={message.plaintext} />
        <MessageField label="Шифртекст" value={message.ciphertext} />
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
      </div>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </article>
  );
}
