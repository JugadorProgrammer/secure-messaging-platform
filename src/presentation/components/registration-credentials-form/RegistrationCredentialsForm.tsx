'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {emailPattern, loginPattern, passwordPattern} from '@/infrastructure/http/signupSchema';
import {signup} from '@/presentation/api/signup';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './RegistrationCredentialsForm.module.css';

type RegistrationFailure = {
  notificationDate: number;
  message: string;
};

type RegistrationCredentialsFormProps = {
  onSuccess: (codes: string[]) => void;
};

export function RegistrationCredentialsForm({onSuccess}: RegistrationCredentialsFormProps) {
  const [login, setLogin] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirmation, setPasswordConfirmation] = useState('');
  const [failure, setFailure] = useState<RegistrationFailure | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

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

  function validateForm(): string | null {
    if (!loginPattern.test(login)) {
      return 'Логин: от 3 до 32 символов, латиница и цифры, начинается с буквы.';
    }

    if (!emailPattern.test(email)) {
      return 'Почта указана неверно.';
    }

    if (!passwordPattern.test(password) || !passwordPattern.test(passwordConfirmation)) {
      return 'Пароль: от 8 до 64 символов, строчная и заглавная буква и цифра.';
    }

    if (password !== passwordConfirmation) {
      return 'Пароли не совпадают.';
    }

    return null;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const errors = validateForm();

    if (errors !== null) {
      setFailure({
        notificationDate: Date.now(),
        message: errors,
      });
      return;
    }

    setIsSubmitting(true);
    setFailure(null);

    const result = await signup({login, email, password});

    setPassword('');
    setPasswordConfirmation('');
    setIsSubmitting(false);

    if (!result.isSuccess) {
      setFailure({
        notificationDate: Date.now(),
        message: result.error.message,
      });
      return;
    }

    onSuccess(result.success);
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
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
