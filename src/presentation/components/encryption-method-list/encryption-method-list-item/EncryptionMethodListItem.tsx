import type {EncryptionMethod} from '@/domain/encryptionMethod';

import styles from './EncryptionMethodListItem.module.css';

type EncryptionMethodListItemProps = {
  method: EncryptionMethod;
  isAvailabilityPending?: boolean;
  onAvailabilityChange?: (enabled: boolean) => void;
};

export function EncryptionMethodListItem({
  method,
  isAvailabilityPending = false,
  onAvailabilityChange,
}: EncryptionMethodListItemProps) {
  const status = method.enabled ? 'включён' : 'выключен';

  return (
    <li className={styles['encryption-method-list-item']}>
      <span className={styles['encryption-method-list-item__name']}>{method.name}</span>
      {onAvailabilityChange !== undefined ? (
        <button
          type="button"
          className={styles['encryption-method-list-item__toggle']}
          aria-pressed={method.enabled}
          disabled={isAvailabilityPending}
          onClick={() => {
            onAvailabilityChange(!method.enabled);
          }}
        >
          {status}
        </button>
      ) : (
        <span className={styles['encryption-method-list-item__status']}>{status}</span>
      )}
    </li>
  );
}
