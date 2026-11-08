'use client';

import {useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';


import {confirmNewPassword} from '@/presentation/api/passwordRecovery';
import {Notification} from '@/presentation/components/notification/Notification';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

import styles from './NewPasswordForm.module.css';

type NewPasswordFormProps = {
  onSuccess: () => void;
};

export function NewPasswordForm({onSuccess}: NewPasswordFormProps) {
  const [password, setPassword] = useState('');
  const {notice, show, dismiss} = useScreenNotice();
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    dismiss();

    const result = await confirmNewPassword(password);

    setPassword('');
    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess();
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
