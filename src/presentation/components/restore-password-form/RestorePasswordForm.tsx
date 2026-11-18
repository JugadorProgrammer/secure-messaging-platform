'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {useRestorePasswordForm} from './useRestorePasswordForm';

import styles from './RestorePasswordForm.module.css';

type RestorePasswordFormProps = {
  onSuccess: (email: string) => void;
};

export function RestorePasswordForm({onSuccess}: RestorePasswordFormProps) {
  const {email, setEmail, isSubmitting, notice, submit} = useRestorePasswordForm({onSuccess});

  function handleEmailChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmail(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
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
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
