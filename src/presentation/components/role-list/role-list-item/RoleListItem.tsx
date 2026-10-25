import type {Role} from '@/domain/role';

import styles from './RoleListItem.module.css';

type RoleListItemProps = {
  role: Role;
};

export function RoleListItem({role}: RoleListItemProps) {
  return (
    <div className={styles['role-list-item']}>
      <span className={styles['role-list-item__name']}>{role.name}</span>
      <span className={styles['role-list-item__description']}>{role.description}</span>
    </div>
  );
}
