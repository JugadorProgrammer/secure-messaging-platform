'use client';

import {useEffect, useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';


import {confirmRestoreCode, restorePassword} from '@/presentation/api/passwordRecovery';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

const RESEND_DELAY_SECONDS = 60;

type UsePasswordRecoveryCodeFormParams = {
  email: string;
  onSuccess: () => void;
};

export function usePasswordRecoveryCodeForm({
  email,
  onSuccess,
}: UsePasswordRecoveryCodeFormParams): {
  confirmationCode: string;
  setConfirmationCode: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  isResendDisabled: boolean;
  resendButtonLabel: string;
  submit: () => Promise<void>;
  resend: () => Promise<void>;
} {
  const [confirmationCode, setConfirmationCode] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [secondsLeft, setSecondsLeft] = useState(RESEND_DELAY_SECONDS);
  const [isResending, setIsResending] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  useEffect(() => {
    if (secondsLeft === 0) {
      return;
    }

    const timerId = window.setTimeout(() => {
      setSecondsLeft((seconds) => seconds - 1);
    }, 1000);

    return () => {
      window.clearTimeout(timerId);
    };
  }, [secondsLeft]);

  const isResendDisabled = isResending || secondsLeft > 0;
  const resendButtonLabel =
    secondsLeft > 0 ? `Отправить код ещё раз через ${secondsLeft} с` : 'Отправить код ещё раз';

  async function submit(): Promise<void> {
    setIsSubmitting(true);
    dismiss();

    const result = await confirmRestoreCode(confirmationCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess();
  }

  async function resend(): Promise<void> {
    setIsResending(true);
    dismiss();

    const result = await restorePassword(email);

    setIsResending(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    setSecondsLeft(RESEND_DELAY_SECONDS);
  }

  return {
    confirmationCode,
    setConfirmationCode,
    isSubmitting,
    notice,
    isResendDisabled,
    resendButtonLabel,
    submit,
    resend,
  };
}
