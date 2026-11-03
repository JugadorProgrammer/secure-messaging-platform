import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {signup} from '@/presentation/api/signup';

import {RegistrationForm} from './RegistrationForm';

const recoveryCodes = ['AB12CD34', 'EF56GH78'];

const meta = {
  title: 'RegistrationForm',
  component: RegistrationForm,
  parameters: {
    layout: 'padded',
    nextjs: {
      appDirectory: true,
    },
  },
  beforeEach() {
    mocked(signup).mockResolvedValue(Result.success(recoveryCodes));
  },
} satisfies Meta<typeof RegistrationForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Credentials: Story = {
  name: 'Регистрация',
};

export const Codes: Story = {
  name: 'Коды восстановления',
  args: {
    initialCodes: ["AB12CD34", "EF56GH78", "EF56GH78", "EF56GH78", "EF56GH78"],
  },
};
