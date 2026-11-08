'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {restorePassword} from '@/presentation/api/passwordRecovery';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseRestorePasswordFormParams = {
  onSuccess: (email: string) => void;
};

export function useRestorePasswordForm({onSuccess}: UseRestorePasswordFormParams): {
  email: string;
  setEmail: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [email, setEmail] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  async function submit(): Promise<void> {
    setIsSubmitting(true);
    dismiss();

    const result = await restorePassword(email);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess(email);
  }

  return {
    email,
    setEmail,
    isSubmitting,
    notice,
    submit,
  };
}
