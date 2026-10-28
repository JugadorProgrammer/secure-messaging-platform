import type {EncryptionMethod} from '@/domain/encryptionMethod';

import styles from './EncryptionMethodListItem.module.css';

type EncryptionMethodListItemProps = {
  method: EncryptionMethod;
};

export function EncryptionMethodListItem({method}: EncryptionMethodListItemProps) {
  return (
    <li className={styles['encryption-method-list-item']}>
      <span className={styles['encryption-method-list-item__name']}>{method.name}</span>
      <span className={styles['encryption-method-list-item__status']}>
        {method.enabled ? 'включён' : 'выключен'}
      </span>
    </li>
  );
}
