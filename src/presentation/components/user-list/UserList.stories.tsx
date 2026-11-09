import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import {UserList} from './UserList';

const users: UserDto[] = [
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

const meta = {
  title: 'UserList',
  component: UserList,
  parameters: {
    layout: 'padded',
  },
  args: {
    users,
  },
} satisfies Meta<typeof UserList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithUsers: Story = {
  name: 'С пользователями',
};

export const Empty: Story = {
  name: 'Пустой',
  args: {
    users: [],
  },
};
