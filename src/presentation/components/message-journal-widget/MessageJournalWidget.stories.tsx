import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import type {Message} from '@/domain/message';

import {Result} from '@/application/Result';

import {getAllMessages} from '@/presentation/api/messages';

import {MessageJournalWidget} from './MessageJournalWidget';

const meta = {
  title: 'MessageJournalWidget',
  component: MessageJournalWidget,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof MessageJournalWidget>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithMessages: Story = {
  name: 'С сообщениями',
  async beforeEach() {
    mocked(getAllMessages).mockResolvedValue(
      Result.success([
        {
          method: 'RSA-32768',
          plaintext: 'Секретное сообщение',
          ciphertext: 'ciphertext-rsa',
          timestamp: '2026-11-08T12:00:00.000Z',
          keys: {
            publicKey: '-----BEGIN PUBLIC KEY-----\nPUBLIC\n-----END PUBLIC KEY-----',
            privateKey: '-----BEGIN RSA PRIVATE KEY-----\nPRIVATE\n-----END RSA PRIVATE KEY-----',
          },
        },
        {
          method: 'Кузнечик',
          plaintext: 'Другое сообщение',
          ciphertext: 'ciphertext-kuznyechik',
          timestamp: '2026-11-08T12:05:00.000Z',
          keys: {
            key: '00112233445566778899aabbccddeeff',
            iv: 'ffeeddccbbaa99887766554433221100',
          },
        },
      ]),
    );
  },
};

export const Empty: Story = {
  name: 'Пустой',
  async beforeEach() {
    mocked(getAllMessages).mockResolvedValue(Result.success([]));
  },
};

export const Loading: Story = {
  name: 'Загрузка',
  async beforeEach() {
    mocked(getAllMessages).mockImplementation(
      () => new Promise<Result<Message[]>>(() => {}),
    );
  },
};

export const Failure: Story = {
  name: 'Ошибка',
  async beforeEach() {
    mocked(getAllMessages).mockResolvedValue(
      Result.error(new Error('Не удалось загрузить журнал сообщений.')),
    );
  },
};
