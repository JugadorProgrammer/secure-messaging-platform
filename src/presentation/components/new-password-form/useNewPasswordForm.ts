'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {confirmNewPassword} from '@/presentation/api/passwordRecovery';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseNewPasswordFormParams = {
  onSuccess: () => void;
};

export function useNewPasswordForm({onSuccess}: UseNewPasswordFormParams): {
  password: string;
  setPassword: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  async function submit(): Promise<void> {
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

  return {
    password,
    setPassword,
    isSubmitting,
    notice,
    submit,
  };
}
