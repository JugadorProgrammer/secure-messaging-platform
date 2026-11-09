'use client';

import {Notification} from '@/presentation/components/notification/Notification';
import {UserList} from '@/presentation/components/user-list/UserList';

import {useUserListPanel} from './useUserListPanel';

import styles from './UserListPanel.module.css';

export function UserListPanel() {
  const {users, isUsersLoadFailed} = useUserListPanel();

  return (
    <div className={styles['user-list-panel']}>
      <h2 className={styles['user-list-panel__title']}>Список пользователей</h2>
      {isUsersLoadFailed ? <div className={styles['user-list-panel__failed']} /> : null}
      {users === null && !isUsersLoadFailed ? (
        <div className={styles['user-list-panel__bars']} aria-busy="true">
          <span className={styles['user-list-panel__bar']} />
          <span className={styles['user-list-panel__bar']} />
          <span className={styles['user-list-panel__bar']} />
        </div>
      ) : null}
      {users !== null ? <UserList users={users} /> : null}
      {isUsersLoadFailed ? (
        <Notification message="Не удалось загрузить список пользователей." />
      ) : null}
    </div>
  );
}
