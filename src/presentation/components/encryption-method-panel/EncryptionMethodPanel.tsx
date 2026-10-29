'use client';

import {EncryptionMethodList} from '@/presentation/components/encryption-method-list/EncryptionMethodList';
import {Notification} from '@/presentation/components/notification/Notification';

import {useEncryptionMethodPanel} from './useEncryptionMethodPanel';

import styles from './EncryptionMethodPanel.module.css';

export function EncryptionMethodPanel() {
  const state = useEncryptionMethodPanel();

  return (
    <div className={styles['encryption-method-panel']}>
      <h2 className={styles['encryption-method-panel__title']}>Методы шифрования</h2>
      {state.isFailed ? (
        <div className={styles['encryption-method-panel__failed']} />
      ) : null}
      {state.methods === null && !state.isFailed ? (
        <div className={styles['encryption-method-panel__bars']} aria-busy="true">
          <span className={styles['encryption-method-panel__bar']} />
          <span className={styles['encryption-method-panel__bar']} />
          <span className={styles['encryption-method-panel__bar']} />
        </div>
      ) : null}
      {state.methods !== null ? <EncryptionMethodList methods={state.methods} /> : null}
      {state.isFailed ? (
        <Notification message="Не удалось загрузить список методов шифрования." />
      ) : null}
    </div>
  );
}
