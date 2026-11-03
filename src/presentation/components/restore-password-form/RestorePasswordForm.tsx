'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {restorePassword} from '@/presentation/api/passwordRecovery';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './RestorePasswordForm.module.css';

type RecoveryFailure = {
  notificationDate: number;
  message: string;
};

type RestorePasswordFormProps = {
  onSuccess: (email: string) => void;
};

export function RestorePasswordForm({onSuccess}: RestorePasswordFormProps) {
  const [email, setEmail] = useState('');
  const [failure, setFailure] = useState<RecoveryFailure | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleEmailChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmail(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    setFailure(null);

    const result = await restorePassword(email);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      setFailure({
        notificationDate: Date.now(),
        message: result.error.message,
      });
      return;
    }

    onSuccess(email);
  }

  return (
    <form className={styles['restore-password-form']} onSubmit={handleSubmit}>
      <label className={styles['restore-password-form__field']}>
        <span className={styles['restore-password-form__label']}>Почта</span>
        <input
          className={styles['restore-password-form__input']}
          name="email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={handleEmailChange}
          required
        />
      </label>
      <button className={styles['restore-password-form__button']} type="submit" disabled={isSubmitting}>
        Отправить
      </button>
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
