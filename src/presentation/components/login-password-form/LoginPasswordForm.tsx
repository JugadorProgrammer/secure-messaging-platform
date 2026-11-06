'use client';

import type {ChangeEvent, FormEvent} from 'react';
import Link from 'next/link';

import {Notification} from '@/presentation/components/notification/Notification';

import {useLoginPasswordForm} from './useLoginPasswordForm';

import styles from './LoginPasswordForm.module.css';

type LoginPasswordFormProps = {
  onSuccess: () => void;
};

export function LoginPasswordForm({onSuccess}: LoginPasswordFormProps) {
  const {
    login,
    setLogin,
    password,
    setPassword,
    isSubmitting,
    notice,
    submit,
  } = useLoginPasswordForm({onSuccess});

  function handleLoginChange(event: ChangeEvent<HTMLInputElement>): void {
    setLogin(event.target.value);
  }

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  return (
    <form className={styles['login-password-form']} onSubmit={handleSubmit}>
      <label className={styles['login-password-form__field']}>
        <span className={styles['login-password-form__label']}>Логин</span>
        <input
          className={styles['login-password-form__input']}
          name="login"
          autoComplete="username"
          value={login}
          onChange={handleLoginChange}
          required
        />
      </label>
      <label className={styles['login-password-form__field']}>
        <span className={styles['login-password-form__label']}>Пароль</span>
        <input
          className={styles['login-password-form__input']}
          name="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={handlePasswordChange}
          required
        />
      </label>
      <button className={styles['login-password-form__submit']} type="submit" disabled={isSubmitting}>
        Войти
      </button>
      <Link className={styles['login-password-form__link']} href="/recovery">
        Восстановление пароля
      </Link>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
