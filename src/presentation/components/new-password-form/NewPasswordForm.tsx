'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {useNewPasswordForm} from './useNewPasswordForm';

import styles from './NewPasswordForm.module.css';

type NewPasswordFormProps = {
  onSuccess: () => void;
};

export function NewPasswordForm({onSuccess}: NewPasswordFormProps) {
  const {password, setPassword, isSubmitting, notice, submit} = useNewPasswordForm({onSuccess});

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  return (
    <form className={styles['new-password-form']} onSubmit={handleSubmit}>
      <label className={styles['new-password-form__field']}>
        <span className={styles['new-password-form__label']}>Новый пароль</span>
        <input
          className={styles['new-password-form__input']}
          name="password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={handlePasswordChange}
          required
        />
      </label>
      <button className={styles['new-password-form__submit']} type="submit" disabled={isSubmitting}>
        Сохранить
      </button>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
