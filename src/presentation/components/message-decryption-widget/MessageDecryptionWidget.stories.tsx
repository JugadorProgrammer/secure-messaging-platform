import type {Meta, StoryObj} from '@storybook/nextjs-vite';
import {mocked} from 'storybook/test';

import {Result} from '@/application/Result';

import {decryptMessage} from '@/presentation/api/decrypt';
import {getAllEncryptionMethods} from '@/presentation/api/encryptionMethods';

import {MessageDecryptionWidget} from './MessageDecryptionWidget';

const meta = {
  title: 'MessageDecryptionWidget',
  component: MessageDecryptionWidget,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof MessageDecryptionWidget>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Form: Story = {
  name: 'Форма',
};

export const SuccessRsa: Story = {
  name: 'Успех RSA',
  async beforeEach() {
    mocked(decryptMessage).mockResolvedValue(
      Result.success({
        plaintext: 'Секретное сообщение',
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
    mocked(decryptMessage).mockResolvedValue(
      Result.success({
        plaintext: 'Секретное сообщение',
      }),
    );
  },
};

export const Failure: Story = {
  name: 'Ошибка',
  async beforeEach() {
    mocked(decryptMessage).mockResolvedValue(
      Result.error(new Error('Не удалось расшифровать сообщение.')),
    );
  },
};
