import styles from './HomeScreen.module.css';

export function HomeScreen() {
  return (
    <main className={styles['home-screen']}>
      <h1 className={styles['home-screen__title']}>Secure Messaging Platform</h1>
    </main>
  );
}
