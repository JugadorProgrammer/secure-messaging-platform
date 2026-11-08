import {ADMIN_ROLE_DESCRIPTION, ADMIN_ROLE_NAME} from '@/domain/role';

import {AllEncryptionMethodsPanel} from '@/presentation/components/all-encryption-methods-panel/AllEncryptionMethodsPanel';

import styles from './AdminScreen.module.css';

export function AdminScreen() {
  return (
    <main className={styles['admin-screen']}>
      <h1 className={styles['admin-screen__title']}>Страница администратора</h1>
      <section className={styles['admin-screen__widget']}>
        <div className={styles['admin-screen__role']}>
          <p className={styles['admin-screen__role-name']}>{ADMIN_ROLE_NAME}</p>
          <p className={styles['admin-screen__role-description']}>{ADMIN_ROLE_DESCRIPTION}</p>
        </div>
      </section>
      <section className={styles['admin-screen__widget']}>
        <AllEncryptionMethodsPanel />
      </section>
    </main>
  );
}
