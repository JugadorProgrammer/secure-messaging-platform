'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {confirmEmailCode} from '@/presentation/api/login';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseLoginEmailCodeFormParams = {
  onSuccess: () => void;
};

export function useLoginEmailCodeForm({onSuccess}: UseLoginEmailCodeFormParams): {
  emailCode: string;
  setEmailCode: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [emailCode, setEmailCode] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  async function submit(): Promise<void> {
    setIsSubmitting(true);
    dismiss();

    const result = await confirmEmailCode(emailCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess();
  }

  return {
    emailCode,
    setEmailCode,
    isSubmitting,
    notice,
    submit,
  };
}
