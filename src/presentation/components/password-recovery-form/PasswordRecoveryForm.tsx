'use client';

import {usePasswordRecoveryForm, type RecoveryStep} from './usePasswordRecoveryForm';

import styles from './PasswordRecoveryForm.module.css';

type PasswordRecoveryFormProps = {
  initialStep?: RecoveryStep;
  initialEmail?: string;
};

export function PasswordRecoveryForm({
  initialStep = 'email',
  initialEmail = '',
}: PasswordRecoveryFormProps) {
  const {stepForm} = usePasswordRecoveryForm({initialStep, initialEmail});

  return (
    <section className={styles['password-recovery-form']}>
      <h1 className={styles['password-recovery-form__title']}>Восстановление пароля</h1>
      {stepForm}
    </section>
  );
}
