'use client';

import {useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import {
  emailPattern,
  loginPattern,
  passwordPattern,
} from '@/infrastructure/http/RegistrationService/registrationRequestSchema';
import {signup} from '@/presentation/api/signup';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseRegistrationCredentialsFormParams = {
  onSuccess: (codes: string[]) => void;
};

export function useRegistrationCredentialsForm({
  onSuccess,
}: UseRegistrationCredentialsFormParams): {
  login: string;
  setLogin: Dispatch<SetStateAction<string>>;
  email: string;
  setEmail: Dispatch<SetStateAction<string>>;
  password: string;
  setPassword: Dispatch<SetStateAction<string>>;
  passwordConfirmation: string;
  setPasswordConfirmation: Dispatch<SetStateAction<string>>;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
} {
  const [login, setLogin] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirmation, setPasswordConfirmation] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();

  function validateForm(): string | null {
    if (!loginPattern.test(login)) {
      return 'Логин: от 3 до 32 символов, латиница и цифры, начинается с буквы.';
    }

    if (!emailPattern.test(email)) {
      return 'Почта указана неверно.';
    }

    if (!passwordPattern.test(password) || !passwordPattern.test(passwordConfirmation)) {
      return 'Пароль: от 8 до 64 символов, строчная и заглавная буква и цифра.';
    }

    if (password !== passwordConfirmation) {
      return 'Пароли не совпадают.';
    }

    return null;
  }

  async function submit(): Promise<void> {
    const error = validateForm();

    if (error !== null) {
      show(error);
      return;
    }

    setIsSubmitting(true);
    dismiss();

    const result = await signup({login, email, password});

    setPassword('');
    setPasswordConfirmation('');
    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    onSuccess(result.success);
  }

  return {
    login,
    setLogin,
    email,
    setEmail,
    password,
    setPassword,
    passwordConfirmation,
    setPasswordConfirmation,
    isSubmitting,
    notice,
    submit,
  };
}
