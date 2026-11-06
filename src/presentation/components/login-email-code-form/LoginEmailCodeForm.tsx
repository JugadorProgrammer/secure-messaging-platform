'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {Result} from '@/application/Result';

import {confirmEmailCode} from '@/presentation/api/login';
import {Notification} from '@/presentation/components/notification/Notification';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

import styles from './LoginEmailCodeForm.module.css';

type LoginEmailCodeFormProps = {
  onSuccess: () => void;
};

export function LoginEmailCodeForm({onSuccess}: LoginEmailCodeFormProps) {
  const [emailCode, setEmailCode] = useState('');
  const {notice, show, dismiss} = useScreenNotice();
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleEmailCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmailCode(event.target.value.replace(/\D/g, '').slice(0, 6));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    dismiss();

    const result = await confirmEmailCode(emailCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(Result.message(result));
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
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
