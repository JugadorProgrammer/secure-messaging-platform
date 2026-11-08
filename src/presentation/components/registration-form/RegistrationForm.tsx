'use client';

import {useRegistrationForm} from './useRegistrationForm';

import styles from './RegistrationForm.module.css';

type RegistrationFormProps = {
  initialCodes?: readonly string[];
};

export function RegistrationForm({initialCodes}: RegistrationFormProps) {
  const {step} = useRegistrationForm({initialCodes});

  return (
    <section className={styles['registration-form']}>
      <h1 className={styles['registration-form__title']}>Регистрация</h1>
      {step}
    </section>
  );
}
