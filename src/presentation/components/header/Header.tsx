import Link from 'next/link';

import styles from './Header.module.css';

type HeaderProps = {
  isAuthorized: boolean;
};

export function Header({isAuthorized}: HeaderProps) {
  return (
    <header className={styles['header']}>
      <Link className={styles['header__title']} href="/">
        Secure Messaging Platform
      </Link>
      <nav className={styles['header__navigation']}>
        {isAuthorized ? null : (
          <Link className={styles['header__link']} href="/login">
            вход
          </Link>
        )}
        <a className={styles['header__link']} href="/mail">
          почта
        </a>
        {isAuthorized ? (
          <a className={styles['header__link']} href="/logout">
            выйти
          </a>
        ) : (
          <Link className={styles['header__link']} href="/registration">
            регистрация
          </Link>
        )}
      </nav>
    </header>
  );
}
