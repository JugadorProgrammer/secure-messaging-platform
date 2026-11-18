import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {getRoles} from '@/presentation/api/roles';
import {createUser} from '@/presentation/api/users';

import {CreateUserForm} from './CreateUserForm';

const recoveryCodes = ['AB12CD34', 'EF56GH78'];

const meta = {
  title: 'CreateUserForm',
  component: CreateUserForm,
  parameters: {
    layout: 'padded',
  },
  beforeEach() {
    mocked(createUser).mockResolvedValue(Result.success(recoveryCodes));
  },
} satisfies Meta<typeof CreateUserForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Form: Story = {
  name: 'Форма',
};

export const Failure: Story = {
  name: 'Ошибка',
  beforeEach() {
    mocked(createUser).mockResolvedValue(Result.error(new Error('Не удалось создать пользователя.')));
  },
};

export const RolesFailed: Story = {
  name: 'Роли не загрузились',
  beforeEach() {
    mocked(getRoles).mockResolvedValue(Result.error(new Error('down')));
  },
};
