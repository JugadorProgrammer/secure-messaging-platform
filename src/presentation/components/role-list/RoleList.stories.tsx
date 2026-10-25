import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import type {Role} from '@/domain/role';

import {RoleList} from './RoleList';

const roles: Role[] = [
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

const meta = {
  title: 'RoleList',
  component: RoleList,
  parameters: {
    layout: 'padded',
  },
  args: {
    roles,
  },
} satisfies Meta<typeof RoleList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithRoles: Story = {
  name: 'С ролями',
};

export const Empty: Story = {
  name: 'Пустой',
  args: {
    roles: [],
  },
};
