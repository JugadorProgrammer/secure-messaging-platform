import type {Role} from '@/domain/role';

export const sampleRoles: Role[] = [
  {
    id: '1',
    name: 'Пользователь',
    description: 'Участник переписки',
  },
  {
    id: '2',
    name: 'Администратор',
    description: 'Управляет ролями',
  },
];
