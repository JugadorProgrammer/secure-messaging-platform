'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {confirmDisposableCode} from '@/presentation/api/login';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseLoginDisposableCodeFormParams = {
  onSuccess: () => void;
};

export function useLoginDisposableCodeForm({onSuccess}: UseLoginDisposableCodeFormParams): {
  disposableCode: string;
  setDisposableCode: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [disposableCode, setDisposableCode] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  async function submit(): Promise<void> {
    setIsSubmitting(true);
    dismiss();

    const result = await confirmDisposableCode(disposableCode);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess();
  }

  return {
    disposableCode,
    setDisposableCode,
    isSubmitting,
    notice,
    submit,
  };
}
