import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {MessageWidget} from './MessageWidget';

const meta = {
  title: 'MessageWidget',
  component: MessageWidget,
  parameters: {
    layout: 'padded',
  },
  args: {
    message: {
      method: 'RSA-32768',
      plaintext: 'Секретное сообщение',
      ciphertext: 'ciphertext-rsa',
      timestamp: '2026-11-08T12:00:00.000Z',
    },
  },
} satisfies Meta<typeof MessageWidget>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  name: 'По умолчанию',
};
