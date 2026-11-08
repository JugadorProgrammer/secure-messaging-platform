'use client';

import {EncryptionMethodList} from '@/presentation/components/encryption-method-list/EncryptionMethodList';
import {Notification} from '@/presentation/components/notification/Notification';

import {useAllEncryptionMethodsPanel} from './useAllEncryptionMethodsPanel';

import styles from './AllEncryptionMethodsPanel.module.css';

export function AllEncryptionMethodsPanel() {
  const {encryptionMethods, isEncryptionMethodsLoadFailed} = useAllEncryptionMethodsPanel();

  return (
    <div className={styles['all-encryption-methods-panel']}>
      <h2 className={styles['all-encryption-methods-panel__title']}>Доступные методы шифрования</h2>
      {isEncryptionMethodsLoadFailed ? (
        <div className={styles['all-encryption-methods-panel__failed']} />
      ) : null}
      {encryptionMethods === null && !isEncryptionMethodsLoadFailed ? (
        <div className={styles['all-encryption-methods-panel__bars']} aria-busy="true">
          <span className={styles['all-encryption-methods-panel__bar']} />
          <span className={styles['all-encryption-methods-panel__bar']} />
          <span className={styles['all-encryption-methods-panel__bar']} />
        </div>
      ) : null}
      {encryptionMethods !== null ? <EncryptionMethodList methods={encryptionMethods} /> : null}
      {isEncryptionMethodsLoadFailed ? (
        <Notification message="Не удалось загрузить список методов шифрования." />
      ) : null}
    </div>
  );
}
