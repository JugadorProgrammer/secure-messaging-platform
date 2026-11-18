'use client';

import {useState} from 'react';
import type {ComponentType} from 'react';
import {useRouter} from 'next/navigation';

import {LoginDisposableCodeForm} from '@/presentation/components/login-disposable-code-form/LoginDisposableCodeForm';
import {LoginEmailCodeForm} from '@/presentation/components/login-email-code-form/LoginEmailCodeForm';
import {LoginPasswordForm} from '@/presentation/components/login-password-form/LoginPasswordForm';

export type LoginStep = 'password' | 'emailCode' | 'disposableCode';

type LoginStepFormProps = {
  onSuccess: () => void;
};

const forms: Record<LoginStep, ComponentType<LoginStepFormProps>> = {
  password: LoginPasswordForm,
  emailCode: LoginEmailCodeForm,
  disposableCode: LoginDisposableCodeForm,
};

const nextStep: Record<LoginStep, LoginStep | null> = {
  password: 'emailCode',
  emailCode: 'disposableCode',
  disposableCode: null,
};

type UseLoginFormParams = {
  initialStep?: LoginStep;
};

export function useLoginForm({initialStep = 'password'}: UseLoginFormParams = {}): {
  StepForm: ComponentType<LoginStepFormProps>;
  handleSuccess: () => void;
} {
  const router = useRouter();
  const [step, setStep] = useState<LoginStep>(initialStep);

  function handleSuccess(): void {
    const following = nextStep[step];

    if (following === null) {
      router.push('/');
      router.refresh();
      return;
    }

    setStep(following);
  }

  return {
    StepForm: forms[step],
    handleSuccess,
  };
}
