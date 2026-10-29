import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {EncryptionMethodListItem} from '@/presentation/components/encryption-method-list/encryption-method-list-item/EncryptionMethodListItem';

import styles from './EncryptionMethodList.module.css';

type EncryptionMethodListProps = {
  methods: EncryptionMethod[];
};

export function EncryptionMethodList({methods}: EncryptionMethodListProps) {
  if (methods.length === 0) {
    return <p className={styles['encryption-method-list']}>Методов шифрования нет</p>;
  }

  return (
    <ul className={styles['encryption-method-list']}>
      {methods.map((method) => (
        <EncryptionMethodListItem key={method.name} method={method} />
      ))}
    </ul>
  );
}
