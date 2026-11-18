import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {fn, mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {encryptMessage} from '@/presentation/api/encrypt';
import {getAllEncryptionMethods} from '@/presentation/api/encryptionMethods';

import {MessageEncryptionWidget} from './MessageEncryptionWidget';

const meta = {
  title: 'MessageEncryptionWidget',
  component: MessageEncryptionWidget,
  parameters: {
    layout: 'padded',
  },
  args: {
    onEncrypted: fn(),
  },
} satisfies Meta<typeof MessageEncryptionWidget>;

export default meta;

type Story = StoryObj<typeof meta>;

export const SuccessRsa: Story = {
  name: 'Успех RSA',
  async beforeEach() {
    mocked(encryptMessage).mockResolvedValue(
      Result.success({
        id: 1,
        user: {id: 1, name: 'alice'},
        method: 'RSA-32768',
        plaintext: 'Секретное сообщение',
        ciphertext: 'ciphertext-rsa',
        timestamp: '2026-11-08T10:40:00.000Z',
        keys: {
          publicKey: '-----BEGIN PUBLIC KEY-----\nPUBLIC\n-----END PUBLIC KEY-----',
          privateKey: '-----BEGIN RSA PRIVATE KEY-----\nPRIVATE\n-----END RSA PRIVATE KEY-----',
        },
      }),
    );
  },
};

export const SuccessKuznyechik: Story = {
  name: 'Успех Кузнечик',
  async beforeEach() {
    mocked(getAllEncryptionMethods).mockResolvedValue(
      Result.success([
        {name: 'RSA-32768', enabled: true},
        {name: 'Кузнечик', enabled: true},
      ]),
    );
    mocked(encryptMessage).mockResolvedValue(
      Result.success({
        id: 2,
        user: {id: 1, name: 'alice'},
        method: 'Кузнечик',
        plaintext: 'Секретное сообщение',
        ciphertext: 'ciphertext-kuznyechik',
        timestamp: '2026-11-08T10:40:00.000Z',
        keys: {
          key: '00112233445566778899aabbccddeeff',
          iv: 'ffeeddccbbaa99887766554433221100',
        },
      }),
    );
  },
};

export const Failure: Story = {
  name: 'Ошибка',
  async beforeEach() {
    mocked(encryptMessage).mockResolvedValue(Result.error(new Error('Не удалось зашифровать сообщение.')));
  },
};
