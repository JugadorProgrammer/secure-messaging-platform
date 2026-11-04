'use client';

import {useState} from 'react';
import {useRouter} from 'next/navigation';

import {NewPasswordForm} from '@/presentation/components/new-password-form/NewPasswordForm';
import {PasswordRecoveryCodeForm} from '@/presentation/components/password-recovery-code-form/PasswordRecoveryCodeForm';
import {RestorePasswordForm} from '@/presentation/components/restore-password-form/RestorePasswordForm';

import styles from './PasswordRecoveryForm.module.css';

type RecoveryStep = 'email' | 'confirmationCode' | 'password';

type PasswordRecoveryFormProps = {
  initialStep?: RecoveryStep;
  initialEmail?: string;
};

export function PasswordRecoveryForm({
  initialStep = 'email',
  initialEmail = '',
}: PasswordRecoveryFormProps) {
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

  return (
    <section className={styles['password-recovery-form']}>
      <h1 className={styles['password-recovery-form__title']}>Восстановление пароля</h1>
      {stepForm}
    </section>
  );
}
