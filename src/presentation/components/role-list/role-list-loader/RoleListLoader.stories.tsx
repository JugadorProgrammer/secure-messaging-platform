import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import type {Role} from '@/domain/role';

import {Result} from '@/application/Result';

import {getRoles} from '@/presentation/api/roles';

import {RoleListLoader} from './RoleListLoader';

const meta = {
  title: 'RoleListLoader',
  component: RoleListLoader,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof RoleListLoader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const List: Story = {
  name: 'Список',
};

export const Empty: Story = {
  name: 'Пустой',
  beforeEach() {
    mocked(getRoles).mockResolvedValue(Result.success([]));
  },
};

export const Loading: Story = {
  name: 'Скелетон',
  beforeEach() {
    mocked(getRoles).mockImplementation(() => new Promise<Result<Role[]>>(() => {}));
  },
};

export const Failed: Story = {
  name: 'Не загрузилось',
  beforeEach() {
    mocked(getRoles).mockResolvedValue(Result.error(new Error('down')));

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
