import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import styles from './UserList.module.css';

type UserListProps = {
  users: UserDto[];
};

export function UserList({users}: UserListProps) {
  if (users.length === 0) {
    return <p className={styles['user-list']}>Пользователей нет</p>;
  }

  return (
    <table className={styles['user-list']}>
      <thead>
        <tr>
          <th className={styles['user-list__heading']}>Идентификатор</th>
          <th className={styles['user-list__heading']}>Логин</th>
          <th className={styles['user-list__heading']}>Почта</th>
          <th className={styles['user-list__heading']}>Роль</th>
        </tr>
      </thead>
      <tbody>
        {users.map((user) => (
          <tr key={user.id} className={styles['user-list__row']}>
            <td className={styles['user-list__cell']}>{user.id}</td>
            <td className={styles['user-list__cell']}>{user.login}</td>
            <td className={styles['user-list__cell']}>{user.email}</td>
            <td className={styles['user-list__cell']}>{user.role}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
