'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {useRegistrationCredentialsForm} from './useRegistrationCredentialsForm';

import styles from './RegistrationCredentialsForm.module.css';

type RegistrationCredentialsFormProps = {
  onSuccess: (codes: string[]) => void;
};

export function RegistrationCredentialsForm({onSuccess}: RegistrationCredentialsFormProps) {
  const {
    login,
    setLogin,
    email,
    setEmail,
    password,
    setPassword,
    passwordConfirmation,
    setPasswordConfirmation,
    isSubmitting,
    notice,
    submit,
  } = useRegistrationCredentialsForm({onSuccess});

  function handleLoginChange(event: ChangeEvent<HTMLInputElement>): void {
    setLogin(event.target.value);
  }

  function handleEmailChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmail(event.target.value);
  }

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  function handlePasswordConfirmationChange(event: ChangeEvent<HTMLInputElement>): void {
    setPasswordConfirmation(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  return (
    <form className={styles['registration-credentials-form']} onSubmit={handleSubmit}>
      <label className={styles['registration-credentials-form__field']}>
        <span className={styles['registration-credentials-form__label']}>Логин</span>
        <input
          className={styles['registration-credentials-form__input']}
          name="login"
          autoComplete="username"
          value={login}
          onChange={handleLoginChange}
          required
        />
        <p className={styles['registration-credentials-form__hint']}>
          От 3 до 32 символов, латиница и цифры, начинается с буквы
        </p>
      </label>
      <label className={styles['registration-credentials-form__field']}>
        <span className={styles['registration-credentials-form__label']}>Почта</span>
        <input
          className={styles['registration-credentials-form__input']}
          name="email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={handleEmailChange}
          required
        />
      </label>
      <label className={styles['registration-credentials-form__field']}>
        <span className={styles['registration-credentials-form__label']}>Пароль</span>
        <input
          className={styles['registration-credentials-form__input']}
          name="password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={handlePasswordChange}
          required
        />
        <p className={styles['registration-credentials-form__hint']}>
          От 8 до 64 символов, строчная и заглавная буква и цифра
        </p>
      </label>
      <label className={styles['registration-credentials-form__field']}>
        <span className={styles['registration-credentials-form__label']}>Подтверждение пароля</span>
        <input
          className={styles['registration-credentials-form__input']}
          name="passwordConfirmation"
          type="password"
          autoComplete="new-password"
          value={passwordConfirmation}
          onChange={handlePasswordConfirmationChange}
          required
        />
      </label>
      <button className={styles['registration-credentials-form__submit']} type="submit" disabled={isSubmitting}>
        Зарегистрироваться
      </button>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
