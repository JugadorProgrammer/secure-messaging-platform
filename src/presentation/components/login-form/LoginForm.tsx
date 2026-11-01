'use client';

import {useState} from 'react';
import type {ComponentType} from 'react';
import {useRouter} from 'next/navigation';

import {LoginEmailCodeForm} from '@/presentation/components/login-email-code-form/LoginEmailCodeForm';
import {LoginPasswordForm} from '@/presentation/components/login-password-form/LoginPasswordForm';

import styles from './LoginForm.module.css';

type LoginStep = 'password' | 'emailCode';

type LoginStepFormProps = {
  onSuccess: () => void;
};

const forms: Record<LoginStep, ComponentType<LoginStepFormProps>> = {
  password: LoginPasswordForm,
  emailCode: LoginEmailCodeForm,
};

const nextStep: Record<LoginStep, LoginStep | null> = {
  password: 'emailCode',
  emailCode: null,
};

type LoginFormProps = {
  initialStep?: LoginStep;
};

export function LoginForm({initialStep = 'password'}: LoginFormProps) {
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

  const StepForm = forms[step];

  return (
    <section className={styles['login-form']}>
      <h1 className={styles['login-form__title']}>Вход</h1>
      <StepForm onSuccess={handleSuccess} />
    </section>
  );
}
