import Link from 'next/link';

import styles from './RecoveryCodeList.module.css';

type RecoveryCodeListProps = {
  codes: readonly string[];
};

export function RecoveryCodeList({codes}: RecoveryCodeListProps) {
  return (
    <div className={styles['recovery-code-list']}>
      <p className={styles['recovery-code-list__hint']}>Сохраните коды восстановления. Они показываются один раз.</p>
      <div className={styles['recovery-code-list__items']}>
        {codes.map((code) => (
          <span className={styles['recovery-code-list_item']} key={code}>{code}</span>
        ))}
      </div>
      <Link className={styles['recovery-code-list__link']} href="/login">
        Перейти к логину
      </Link>
    </div>
  );
}
