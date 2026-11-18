import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import {Result} from '@/application/Result';

import {getUsers} from '@/presentation/api/users';

import {UserListPanel} from './UserListPanel';

const meta = {
  title: 'UserListPanel',
  component: UserListPanel,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof UserListPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const List: Story = {
  name: 'Список',
};

export const Empty: Story = {
  name: 'Пустой',
  beforeEach() {
    mocked(getUsers).mockResolvedValue(Result.success([]));
  },
};

export const Loading: Story = {
  name: 'Скелетон',
  beforeEach() {
    mocked(getUsers).mockImplementation(() => new Promise<Result<UserDto[]>>(() => {}));
  },
};

export const Failed: Story = {
  name: 'Не загрузилось',
  beforeEach() {
    mocked(getUsers).mockResolvedValue(Result.error(new Error('down')));

    const originalSetTimeout = window.setTimeout;

    window.setTimeout = ((handler: TimerHandler, timeout?: number) => {
      const delay = typeof timeout === 'number' && timeout >= 1000 ? 0 : timeout;

      return originalSetTimeout(handler, delay);
    }) as typeof window.setTimeout;

    return () => {
      window.setTimeout = originalSetTimeout;
    };
  },
};
