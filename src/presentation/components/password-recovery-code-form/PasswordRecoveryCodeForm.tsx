'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {usePasswordRecoveryCodeForm} from './usePasswordRecoveryCodeForm';

import styles from './PasswordRecoveryCodeForm.module.css';

type PasswordRecoveryCodeFormProps = {
  email: string;
  onSuccess: () => void;
};

export function PasswordRecoveryCodeForm({email, onSuccess}: PasswordRecoveryCodeFormProps) {
  const {
    confirmationCode,
    setConfirmationCode,
    isSubmitting,
    notice,
    isResendDisabled,
    resendButtonLabel,
    submit,
    resend,
  } = usePasswordRecoveryCodeForm({email, onSuccess});

  function handleConfirmationCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setConfirmationCode(event.target.value.replace(/\D/g, '').slice(0, 6));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  async function handleResend(): Promise<void> {
    await resend();
  }

  return (
    <form className={styles['password-recovery-code-form']} onSubmit={handleSubmit}>
      <label className={styles['password-recovery-code-form__field']}>
        <span className={styles['password-recovery-code-form__label']}>Код подтверждения</span>
        <input
          className={styles['password-recovery-code-form__input']}
          name="confirmationCode"
          inputMode="numeric"
          autoComplete="one-time-code"
          value={confirmationCode}
          onChange={handleConfirmationCodeChange}
          pattern="\d{6}"
          minLength={6}
          maxLength={6}
          required
        />
        <p className={styles['password-recovery-code-form__hint']}>На почту отправлен код из 6 цифр</p>
      </label>
      <button className={styles['password-recovery-code-form__button']} type="submit" disabled={isSubmitting}>
        Подтвердить
      </button>
      <button
        className={styles['password-recovery-code-form__button']}
        type="button"
        disabled={isResendDisabled}
        onClick={handleResend}
      >
        {resendButtonLabel}
      </button>
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
