import type {UserDto} from '@/application/IUserService/UserService/UserDto';

export const sampleUsers: UserDto[] = [
  {
    id: 1,
    login: 'alice',
    email: 'alice@example.com',
    role: 'admin',
  },
  {
    id: 2,
    login: 'bob',
    email: 'bob@example.com',
    role: 'user',
  },
];
