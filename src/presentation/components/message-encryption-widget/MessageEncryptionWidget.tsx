'use client';

import type {ChangeEvent, FormEvent} from 'react';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

import {Notification} from '@/presentation/components/notification/Notification';

import {useMessageEncryptionWidget} from './useMessageEncryptionWidget';

import styles from './MessageEncryptionWidget.module.css';

type MessageEncryptionWidgetProps = {
  onEncrypted: (message: MessageDto) => void;
};

export function MessageEncryptionWidget({onEncrypted}: MessageEncryptionWidgetProps) {
  const {
    plaintext,
    setPlaintext,
    method,
    setMethod,
    methods,
    isMethodsFailed,
    isSubmitting,
    notice,
    submit,
  } = useMessageEncryptionWidget({onEncrypted});

  function handlePlaintextChange(event: ChangeEvent<HTMLTextAreaElement>): void {
    setPlaintext(event.target.value);
  }

  function handleMethodChange(event: ChangeEvent<HTMLSelectElement>): void {
    setMethod(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  const hasMethods = methods !== null && methods.length > 0 && method !== null;

  return (
    <form className={styles['message-encryption-widget']} onSubmit={handleSubmit}>
      <h2 className={styles['message-encryption-widget__title']}>Шифрование сообщения</h2>
      <label className={styles['message-encryption-widget__textarea-field']}>
        <span className={styles['message-encryption-widget__label']}>Исходный текст</span>
        <textarea
          className={styles['message-encryption-widget__textarea']}
          name="plaintext"
          value={plaintext}
          onChange={handlePlaintextChange}
          rows={5}
          required
        />
      </label>
      <label className={styles['message-encryption-widget__field']}>
        <span className={styles['message-encryption-widget__label']}>Метод шифрования</span>
        <select
          className={styles['message-encryption-widget__input']}
          name="method"
          value={method ?? ''}
          onChange={handleMethodChange}
          disabled={!hasMethods}
          required
        >
          {hasMethods ? (
            methods.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))
          ) : (
            <option value="">
              {methods === null && !isMethodsFailed
                ? 'Загрузка…'
                : 'Нет доступных методов'}
            </option>
          )}
        </select>
      </label>
      <button
        className={styles['message-encryption-widget__submit']}
        type="submit"
        disabled={isSubmitting || !hasMethods}
      >
        Зашифровать
      </button>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
