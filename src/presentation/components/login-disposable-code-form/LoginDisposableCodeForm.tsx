'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {confirmDisposableCode} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './LoginDisposableCodeForm.module.css';

type LoginFailure = {
  notificationDate: number;
  message: string;
};

type LoginDisposableCodeFormProps = {
  onSuccess: () => void;
};

export function LoginDisposableCodeForm({onSuccess}: LoginDisposableCodeFormProps) {
  const [disposableCode, setDisposableCode] = useState('');
  const [failure, setFailure] = useState<LoginFailure | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleDisposableCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setDisposableCode(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    setFailure(null);

    const result = await confirmDisposableCode(disposableCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      setFailure({
        notificationDate: Date.now(),
        message: result.error.message,
      });
      return;
    }

    onSuccess();
  }

  return (
    <form className={styles['login-disposable-code-form']} onSubmit={handleSubmit}>
      <label className={styles['login-disposable-code-form__field']}>
        <span className={styles['login-disposable-code-form__label']}>Код восстановления</span>
        <input
          className={styles['login-disposable-code-form__input']}
          name="disposableCode"
          autoComplete="off"
          value={disposableCode}
          onChange={handleDisposableCodeChange}
          required
        />
        <p className={styles['login-disposable-code-form__hint']}>Код из списка, полученного при регистрации</p>
      </label>
      <button className={styles['login-disposable-code-form__submit']} type="submit" disabled={isSubmitting}>
        Подтвердить
      </button>
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
