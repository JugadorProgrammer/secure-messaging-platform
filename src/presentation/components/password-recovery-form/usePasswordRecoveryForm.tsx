'use client';

import {useState} from 'react';
import type {ReactElement} from 'react';
import {useRouter} from 'next/navigation';

import {NewPasswordForm} from '@/presentation/components/new-password-form/NewPasswordForm';
import {PasswordRecoveryCodeForm} from '@/presentation/components/password-recovery-code-form/PasswordRecoveryCodeForm';
import {RestorePasswordForm} from '@/presentation/components/restore-password-form/RestorePasswordForm';

export type RecoveryStep = 'email' | 'confirmationCode' | 'password';

type UsePasswordRecoveryFormParams = {
  initialStep?: RecoveryStep;
  initialEmail?: string;
};

export function usePasswordRecoveryForm({
  initialStep = 'email',
  initialEmail = '',
}: UsePasswordRecoveryFormParams = {}): {
  stepForm: ReactElement;
} {
  const router = useRouter();
  const [step, setStep] = useState<RecoveryStep>(initialStep);
  const [email, setEmail] = useState(initialEmail);

  function handleEmailSuccess(sentEmail: string): void {
    setEmail(sentEmail);
    setStep('confirmationCode');
  }

  function handleCodeSuccess(): void {
    setStep('password');
  }

  function handlePasswordSuccess(): void {
    router.push('/login');
    router.refresh();
  }

  const stepForm = {
    email: <RestorePasswordForm onSuccess={handleEmailSuccess} />,
    confirmationCode: <PasswordRecoveryCodeForm email={email} onSuccess={handleCodeSuccess} />,
    password: <NewPasswordForm onSuccess={handlePasswordSuccess} />,
  }[step];

  return {stepForm};
}
