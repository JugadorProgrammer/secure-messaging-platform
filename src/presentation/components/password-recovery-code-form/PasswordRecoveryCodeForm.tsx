'use client';

import {useEffect, useState} from 'react';
import type {ChangeEvent, FormEvent} from 'react';

import {confirmRestoreCode, restorePassword} from '@/presentation/api/passwordRecovery';
import {Notification} from '@/presentation/components/notification/Notification';

import styles from './PasswordRecoveryCodeForm.module.css';

const RESEND_DELAY_SECONDS = 60;

type RecoveryFailure = {
  notificationDate: number;
  message: string;
};

type PasswordRecoveryCodeFormProps = {
  email: string;
  onSuccess: () => void;
};

export function PasswordRecoveryCodeForm({email, onSuccess}: PasswordRecoveryCodeFormProps) {
  const [confirmationCode, setConfirmationCode] = useState('');
  const [failure, setFailure] = useState<RecoveryFailure | null>(null);
  const [isSending, setIsSending] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [resendDelaySeconds, setResendDelaySeconds] = useState(RESEND_DELAY_SECONDS);

  useEffect(() => {
    if (resendDelaySeconds === 0) {
      return;
    }

    const timerId = window.setTimeout(() => {
      setResendDelaySeconds((seconds) => seconds - 1);
    }, 1000);

    return () => {
      window.clearTimeout(timerId);
    };
  }, [resendDelaySeconds]);

  function handleConfirmationCodeChange(event: ChangeEvent<HTMLInputElement>): void {
    setConfirmationCode(event.target.value.replace(/\D/g, '').slice(0, 6));
  }

  async function handleResend(): Promise<void> {
    setIsSending(true);
    setFailure(null);

    const result = await restorePassword(email);

    setIsSending(false);

    if (!result.isSuccess) {
      setFailure({
        notificationDate: Date.now(),
        message: result.error.message,
      });
      return;
    }

    setResendDelaySeconds(RESEND_DELAY_SECONDS);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setIsSubmitting(true);
    setFailure(null);

    const result = await confirmRestoreCode(confirmationCode);

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

  const resendButtonLabel = resendDelaySeconds > 0
    ? `Отправить код ещё раз через ${resendDelaySeconds} с`
    : 'Отправить код ещё раз';

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
        disabled={isSending || resendDelaySeconds > 0}
        onClick={handleResend}
      >
        {resendButtonLabel}
      </button>
      {failure !== null ? (
        <Notification key={failure.notificationDate} message={failure.message} />
      ) : null}
    </form>
  );
}
