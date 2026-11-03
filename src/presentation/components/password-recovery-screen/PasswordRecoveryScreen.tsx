import {PasswordRecoveryForm} from '@/presentation/components/password-recovery-form/PasswordRecoveryForm';

import styles from './PasswordRecoveryScreen.module.css';

export function PasswordRecoveryScreen() {
  return (
    <main className={styles['password-recovery-screen']}>
      <PasswordRecoveryForm />
    </main>
  );
}
