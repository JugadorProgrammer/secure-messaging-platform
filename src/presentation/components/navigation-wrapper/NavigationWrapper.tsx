import type {ReactNode} from 'react';

import styles from './NavigationWrapper.module.css';

type NavigationWrapperProps = {
  children?: ReactNode;
};

export function NavigationWrapper({children}: NavigationWrapperProps) {
  return <nav className={styles['navigation-wrapper']}>{children}</nav>;
}
