'use client';

import {useState} from 'react';

import {RecoveryCodeList} from '@/presentation/components/recovery-code-list/RecoveryCodeList';
import {RegistrationCredentialsForm} from '@/presentation/components/registration-credentials-form/RegistrationCredentialsForm';

import styles from './RegistrationForm.module.css';

type RegistrationFormProps = {
  initialCodes?: readonly string[];
};

export function RegistrationForm({initialCodes}: RegistrationFormProps) {
  const [codes, setCodes] = useState<readonly string[] | null>(initialCodes ?? null);

  const step = codes === null
    ? <RegistrationCredentialsForm onSuccess={setCodes} />
    : <RecoveryCodeList codes={codes} />;

  return (
    <section className={styles['registration-form']}>
      <h1 className={styles['registration-form__title']}>Регистрация</h1>
      {step}
    </section>
  );
}
