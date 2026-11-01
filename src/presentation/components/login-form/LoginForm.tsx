'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';
import Link from 'next/link';
import {useRouter} from 'next/navigation';

import {login as sendLogin} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './LoginForm.module.css';

type LoginFailure = {
  notificationDate: number;
  message: string;
};

export function LoginForm() {
  const router = useRouter();
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

    router.push('/');
    router.refresh();
  }

  return (
    <form className={styles['login-form']} onSubmit={handleSubmit}>
      <h1 className={styles['login-form__title']}>Вход</h1>
      <label className={styles['login-form__field']}>
        <span className={styles['login-form__label']}>Логин</span>
        <input
          className={styles['login-form__input']}
          name="login"
          autoComplete="username"
          value={login}
          onChange={handleLoginChange}
          required
        />
      </label>
      <label className={styles['login-form__field']}>
        <span className={styles['login-form__label']}>Пароль</span>
        <input
          className={styles['login-form__input']}
          name="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={handlePasswordChange}
          required
        />
      </label>
      <button className={styles['login-form__submit']} type="submit" disabled={isSubmitting}>
        Войти
      </button>
      <Link className={styles['login-form__link']} href="/recovery">
        Восстановление пароля
      </Link>
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
