import {LoginForm} from '@/presentation/components/login-form/LoginForm';

import styles from './LoginScreen.module.css';

export function LoginScreen() {
  return (
    <main className={styles['login-screen']}>
      <LoginForm />
    </main>
  );
}
