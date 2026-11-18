'use client';

import {EncryptionMethodList} from '@/presentation/components/encryption-method-list/EncryptionMethodList';
import {Notification} from '@/presentation/components/notification/Notification';

import {useEncryptionMethodPanel} from './useEncryptionMethodPanel';

import styles from './EncryptionMethodPanel.module.css';

export function EncryptionMethodPanel() {
  const {encryptionMethods, isEncryptionMethodsLoadFailed} = useEncryptionMethodPanel();

  return (
    <div className={styles['encryption-method-panel']}>
      <h2 className={styles['encryption-method-panel__title']}>Методы шифрования</h2>
      {isEncryptionMethodsLoadFailed ? (
        <div className={styles['encryption-method-panel__failed']} />
      ) : null}
      {encryptionMethods === null && !isEncryptionMethodsLoadFailed ? (
        <div className={styles['encryption-method-panel__bars']} aria-busy="true">
          <span className={styles['encryption-method-panel__bar']} />
          <span className={styles['encryption-method-panel__bar']} />
          <span className={styles['encryption-method-panel__bar']} />
        </div>
      ) : null}
      {encryptionMethods !== null ? <EncryptionMethodList methods={encryptionMethods} /> : null}
      {isEncryptionMethodsLoadFailed ? (
        <Notification message="Не удалось загрузить список методов шифрования." />
      ) : null}
    </div>
  );
}
