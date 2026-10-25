import type {Role} from '@/domain/role';

import {RoleListItem} from '@/presentation/components/role-list/role-list-item/RoleListItem';

import styles from './RoleList.module.css';

type RoleListProps = {
  roles: Role[];
};

export function RoleList({roles}: RoleListProps) {
  if (roles.length === 0) {
    return <p className={styles['role-list']}>Ролей нет</p>;
  }

  return (
    <ul className={styles['role-list']}>
      {roles.map((role) => (
        <RoleListItem key={role.id} role={role} />
      ))}
    </ul>
  );
}
