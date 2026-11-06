'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {Result} from '@/application/Result';

import {confirmDisposableCode} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

import styles from './LoginDisposableCodeForm.module.css';

type LoginDisposableCodeFormProps = {
  onSuccess: () => void;
};

export function LoginDisposableCodeForm({onSuccess}: LoginDisposableCodeFormProps) {
  const [disposableCode, setDisposableCode] = useState('');
  const {notice, show, dismiss} = useScreenNotice();
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleDisposableCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setDisposableCode(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    dismiss();

    const result = await confirmDisposableCode(disposableCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(Result.message(result));
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
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
