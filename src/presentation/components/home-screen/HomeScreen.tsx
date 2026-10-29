import {EnabledEncryptionMethodsWrapper} from '@/presentation/components/enabled-encryption-methods-wrapper/EnabledEncryptionMethodsWrapper';
import {NavigationPanel} from '@/presentation/components/navigation-panel/NavigationPanel';
import {NavigationWrapper} from '@/presentation/components/navigation-wrapper/NavigationWrapper';
import {RoleListLoader} from '@/presentation/components/role-list/role-list-loader/RoleListLoader';

import styles from './HomeScreen.module.css';

export function HomeScreen() {
  return (
    <main className={styles['home-screen']}>
      <h1 className={styles['home-screen__title']}>Secure Messaging Platform</h1>
      <NavigationWrapper>
        <NavigationPanel />
      </NavigationWrapper>
      <RoleListLoader />
      <EnabledEncryptionMethodsWrapper />
    </main>
  );
}
