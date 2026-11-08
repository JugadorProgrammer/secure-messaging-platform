export const ADMIN_ROLE = 'admin';

export const ADMIN_ROLE_NAME = 'Администратор';

export const ADMIN_ROLE_DESCRIPTION =
  'Управляет методами шифрования и пользователями, просматривает журнал сообщений всех пользователей.';

export type Role = {
  id: number;
  name: string;
  description: string;
};
