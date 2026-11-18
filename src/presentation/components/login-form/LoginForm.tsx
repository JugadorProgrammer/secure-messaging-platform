'use client';

import {useLoginForm, type LoginStep} from './useLoginForm';

import styles from './LoginForm.module.css';

type LoginFormProps = {
  initialStep?: LoginStep;
};

export function LoginForm({initialStep = 'password'}: LoginFormProps) {
  const {StepForm, handleSuccess} = useLoginForm({initialStep});

  return (
    <section className={styles['login-form']}>
      <h1 className={styles['login-form__title']}>Вход</h1>
      <StepForm onSuccess={handleSuccess} />
    </section>
  );
}
