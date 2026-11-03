import {RegistrationForm} from '@/presentation/components/registration-form/RegistrationForm';

import styles from './RegistrationScreen.module.css';

export function RegistrationScreen() {
  return (
    <main className={styles['registration-screen']}>
      <RegistrationForm />
    </main>
  );
}
