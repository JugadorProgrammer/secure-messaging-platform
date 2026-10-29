import clsx from 'clsx';
import Link from 'next/link';

import styles from './NavigationPanel.module.css';

export function NavigationPanel() {
  return (
    <div className={styles['navigation-panel']}>
      <Link
        className={clsx(
          styles['navigation-panel__link'],
          styles['navigation-panel__link--primary'],
        )}
        href="/login"
      >
        войти
      </Link>
      <a className={styles['navigation-panel__link']} href="/mail">
        завести почту
      </a>
      <Link className={styles['navigation-panel__link']} href="/registration">
        создать учётную запись
      </Link>
    </div>
  );
}
