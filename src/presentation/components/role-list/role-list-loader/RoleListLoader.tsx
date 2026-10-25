'use client';

import clsx from 'clsx';

import {Notification} from '@/presentation/components/notification/Notification';
import {RoleList} from '@/presentation/components/role-list/RoleList';

import {useRoleListLoader} from './useRoleListLoader';

import styles from './RoleListLoader.module.css';

export function RoleListLoader() {
  const state = useRoleListLoader();

  if (state.isFailed) {
    return (
      <>
        <div
          className={clsx(
            styles['role-list-loader'],
            styles['role-list-loader--failed'],
          )}
        />
        <Notification message="Не удалось загрузить список ролей." />
      </>
    );
  }

  if (state.roles === null) {
    return (
      <div className={styles['role-list-loader']} aria-busy="true">
        <span className={styles['role-list-loader__bar']} />
        <span className={styles['role-list-loader__bar']} />
        <span className={styles['role-list-loader__bar']} />
      </div>
    );
  }

  return <RoleList roles={state.roles} />;
}
