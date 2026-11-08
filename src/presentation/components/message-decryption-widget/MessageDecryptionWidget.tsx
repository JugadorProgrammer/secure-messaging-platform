'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {useMessageDecryptionWidget} from './useMessageDecryptionWidget';

import styles from './MessageDecryptionWidget.module.css';

export function MessageDecryptionWidget() {
  const {
    ciphertext,
    setCiphertext,
    privateKey,
    setPrivateKey,
    key,
    setKey,
    method,
    setMethod,
    methods,
    isMethodsFailed,
    isSubmitting,
    plaintext,
    notice,
    submit,
  } = useMessageDecryptionWidget();

  function handleCiphertextChange(event: ChangeEvent<HTMLTextAreaElement>): void {
    setCiphertext(event.target.value);
  }

  function handlePrivateKeyChange(event: ChangeEvent<HTMLTextAreaElement>): void {
    setPrivateKey(event.target.value);
  }

  function handleKeyChange(event: ChangeEvent<HTMLInputElement>): void {
    setKey(event.target.value);
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
    <form className={styles['message-decryption-widget']} onSubmit={handleSubmit}>
      <h2 className={styles['message-decryption-widget__title']}>Дешифрование сообщения</h2>
      <label className={styles['message-decryption-widget__field']}>
        <span className={styles['message-decryption-widget__label']}>Метод шифрования</span>
        <select
          className={styles['message-decryption-widget__input']}
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
      <label className={styles['message-decryption-widget__field']}>
        <span className={styles['message-decryption-widget__label']}>Шифртекст</span>
        <textarea
          className={styles['message-decryption-widget__input']}
          name="ciphertext"
          value={ciphertext}
          onChange={handleCiphertextChange}
          rows={5}
          required
        />
      </label>
      {method === 'RSA-32768' ? (
        <label className={styles['message-decryption-widget__field']}>
          <span className={styles['message-decryption-widget__label']}>Private key (PEM)</span>
          <textarea
            className={styles['message-decryption-widget__input']}
            name="privateKey"
            value={privateKey}
            onChange={handlePrivateKeyChange}
            rows={5}
            required
          />
        </label>
      ) : null}
      {method === 'Кузнечик' ? (
        <label className={styles['message-decryption-widget__field']}>
          <span className={styles['message-decryption-widget__label']}>Ключ</span>
          <input
            className={styles['message-decryption-widget__input']}
            name="key"
            value={key}
            onChange={handleKeyChange}
            required
          />
        </label>
      ) : null}
      <button
        className={styles['message-decryption-widget__submit']}
        type="submit"
        disabled={isSubmitting || !hasMethods}
      >
        Расшифровать
      </button>
      {plaintext !== null ? (
        <div className={styles['message-decryption-widget__field']}>
          <span className={styles['message-decryption-widget__label']}>Исходный текст</span>
          <p className={styles['message-decryption-widget__plaintext']}>{plaintext}</p>
        </div>
      ) : null}
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
