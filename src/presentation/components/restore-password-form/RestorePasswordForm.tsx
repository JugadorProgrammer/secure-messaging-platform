'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {Result} from '@/application/Result';

import {restorePassword} from '@/presentation/api/passwordRecovery';
import {Notification} from '@/presentation/components/notification/Notification';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

import styles from './RestorePasswordForm.module.css';

type RestorePasswordFormProps = {
  onSuccess: (email: string) => void;
};

export function RestorePasswordForm({onSuccess}: RestorePasswordFormProps) {
  const [email, setEmail] = useState('');
  const {notice, show, dismiss} = useScreenNotice();
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleEmailChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmail(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    dismiss();

    const result = await restorePassword(email);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(Result.message(result));
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
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
