import Link from 'next/link';

import styles from './Header.module.css';

type HeaderProps = {
  isAuthorized: boolean;
  isAdmin?: boolean;
};

export function Header({isAuthorized, isAdmin = false}: HeaderProps) {
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
        {isAdmin ? (
          <Link className={styles['header__link']} href="/admin">
            страница администратора
          </Link>
        ) : null}
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
