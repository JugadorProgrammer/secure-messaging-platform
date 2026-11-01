import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {confirmEmailCode, login} from '@/presentation/api/login';

import {LoginForm} from './LoginForm';

const meta = {
  title: 'LoginForm',
  component: LoginForm,
  parameters: {
    layout: 'padded',
    nextjs: {
      appDirectory: true,
    },
  },
  beforeEach() {
    mocked(login).mockResolvedValue(Result.success(undefined));
    mocked(confirmEmailCode).mockResolvedValue(Result.success(undefined));
  },
} satisfies Meta<typeof LoginForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Password: Story = {
  name: 'Первый фактор',
};

export const EmailCode: Story = {
  name: 'Второй фактор',
  args: {
    initialStep: 'emailCode',
  },
};
