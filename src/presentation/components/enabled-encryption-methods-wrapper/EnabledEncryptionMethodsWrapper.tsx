import type {ReactNode} from 'react';

import styles from './EnabledEncryptionMethodsWrapper.module.css';

type EnabledEncryptionMethodsWrapperProps = {
  children?: ReactNode;
};

export function EnabledEncryptionMethodsWrapper({children}: EnabledEncryptionMethodsWrapperProps) {
  return (
    <section className={styles['enabled-encryption-methods-wrapper']}>{children}</section>
  );
}
