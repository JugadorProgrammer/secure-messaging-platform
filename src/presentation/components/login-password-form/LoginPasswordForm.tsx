'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';
import Link from 'next/link';

import {login as sendLogin} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './LoginPasswordForm.module.css';

type LoginFailure = {
  notificationDate: number;
  message: string;
};

type LoginPasswordFormProps = {
  onSuccess: () => void;
};

export function LoginPasswordForm({onSuccess}: LoginPasswordFormProps) {
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [failure, setFailure] = useState<LoginFailure | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleLoginChange(event: ChangeEvent<HTMLInputElement>): void {
    setLogin(event.target.value);
  }

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    setFailure(null);

    const result = await sendLogin({login, password});

    setPassword('');
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
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
