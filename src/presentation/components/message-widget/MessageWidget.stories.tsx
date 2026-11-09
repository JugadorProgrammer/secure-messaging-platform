import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';

import {MessageWidget} from './MessageWidget';

const rsaMessage: MessageDto = {
  id: 1,
  user: {id: 1, name: 'alice'},
  method: 'RSA-32768',
  plaintext: 'Секретное сообщение',
  ciphertext: 'ciphertext-rsa',
  timestamp: '2026-11-08T12:00:00.000Z',
  keys: {
    publicKey: '-----BEGIN PUBLIC KEY-----\nPUBLIC\n-----END PUBLIC KEY-----',
    privateKey: '-----BEGIN RSA PRIVATE KEY-----\nPRIVATE\n-----END RSA PRIVATE KEY-----',
  },
};

const kuznyechikMessage: MessageDto = {
  id: 2,
  user: {id: 2, name: 'bob'},
  method: 'Кузнечик',
  plaintext: 'Другое сообщение',
  ciphertext: 'ciphertext-kuznyechik',
  timestamp: '2026-11-08T12:05:00.000Z',
  keys: {
    key: '00112233445566778899aabbccddeeff',
    iv: 'ffeeddccbbaa99887766554433221100',
  },
};

const meta = {
  title: 'MessageWidget',
  component: MessageWidget,
  parameters: {
    layout: 'padded',
  },
  args: {
    message: rsaMessage,
  },
} satisfies Meta<typeof MessageWidget>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Rsa: Story = {
  name: 'RSA',
};

export const Kuznyechik: Story = {
  name: 'Кузнечик',
  args: {
    message: kuznyechikMessage,
  },
};
