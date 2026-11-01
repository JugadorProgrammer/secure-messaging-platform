'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {confirmEmailCode} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './LoginEmailCodeForm.module.css';

type LoginFailure = {
  notificationDate: number;
  message: string;
};

type LoginEmailCodeFormProps = {
  onSuccess: () => void;
};

export function LoginEmailCodeForm({onSuccess}: LoginEmailCodeFormProps) {
  const [emailCode, setEmailCode] = useState('');
  const [failure, setFailure] = useState<LoginFailure | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleEmailCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmailCode(event.target.value.replace(/\D/g, '').slice(0, 6));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    setFailure(null);

    const result = await confirmEmailCode(emailCode);

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
    <form className={styles['login-email-code-form']} onSubmit={handleSubmit}>
      <label className={styles['login-email-code-form__field']}>
        <span className={styles['login-email-code-form__label']}>Код из письма</span>
        <input
          className={styles['login-email-code-form__input']}
          name="emailCode"
          inputMode="numeric"
          autoComplete="one-time-code"
          value={emailCode}
          onChange={handleEmailCodeChange}
          pattern="\d{6}"
          minLength={6}
          maxLength={6}
          required
        />
        <p className={styles['login-email-code-form__hint']}>На почту отправлен код из 6 цифр</p>
      </label>
      <button className={styles['login-email-code-form__submit']} type="submit" disabled={isSubmitting}>
        Подтвердить
      </button>
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
