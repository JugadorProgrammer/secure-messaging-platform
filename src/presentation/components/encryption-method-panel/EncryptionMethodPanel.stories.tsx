import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {Result} from '@/application/Result';

import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';

import {EncryptionMethodPanel} from './EncryptionMethodPanel';

const meta = {
  title: 'EncryptionMethodPanel',
  component: EncryptionMethodPanel,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof EncryptionMethodPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const List: Story = {
  name: 'Список',
};

export const Empty: Story = {
  name: 'Пустой',
  beforeEach() {
    mocked(getEnabledEncryptionMethods).mockResolvedValue(Result.success([]));
  },
};

export const Loading: Story = {
  name: 'Скелетон',
  beforeEach() {
    mocked(getEnabledEncryptionMethods).mockImplementation(
      () => new Promise<Result<EncryptionMethod[]>>(() => {}),
    );
  },
};

export const Failed: Story = {
  name: 'Не загрузилось',
  beforeEach() {
    mocked(getEnabledEncryptionMethods).mockResolvedValue(Result.error(new Error('down')));

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
