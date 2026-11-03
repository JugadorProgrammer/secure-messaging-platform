import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {confirmNewPassword, confirmRestoreCode, restorePassword} from '@/presentation/api/passwordRecovery';

import {PasswordRecoveryForm} from './PasswordRecoveryForm';

const meta = {
  title: 'PasswordRecoveryForm',
  component: PasswordRecoveryForm,
  parameters: {
    layout: 'padded',
    nextjs: {
      appDirectory: true,
    },
  },
  beforeEach() {
    mocked(restorePassword).mockResolvedValue(Result.success(undefined));
    mocked(confirmRestoreCode).mockResolvedValue(Result.success(undefined));
    mocked(confirmNewPassword).mockResolvedValue(Result.success(undefined));
  },
} satisfies Meta<typeof PasswordRecoveryForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Email: Story = {
  name: 'Запрос письма',
};

export const ConfirmationCode: Story = {
  name: 'Код подтверждения',
  args: {
    initialStep: 'confirmationCode',
    initialEmail: 'user@example.com',
  },
};

export const Password: Story = {
  name: 'Новый пароль',
  args: {
    initialStep: 'password',
  },
};
