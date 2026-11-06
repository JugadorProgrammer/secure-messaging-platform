'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {Result} from '@/application/Result';

import {login as sendLogin} from '@/presentation/api/login';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseLoginPasswordFormParams = {
  onSuccess: () => void;
};

export function useLoginPasswordForm({onSuccess}: UseLoginPasswordFormParams): {
  login: string;
  setLogin: Dispatch<SetStateAction<string>>;
  password: string;
  setPassword: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  async function submit(): Promise<void> {
    setIsSubmitting(true);
    dismiss();

    const result = await sendLogin({login, password});

    setPassword('');
    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(Result.message(result));
      return;
    }

    onSuccess();
  }

  return {
    login,
    setLogin,
    password,
    setPassword,
    isSubmitting,
    notice,
    submit,
  };
}
