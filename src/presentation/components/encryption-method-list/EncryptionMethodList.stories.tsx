import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {EncryptionMethodList} from './EncryptionMethodList';

const methods: EncryptionMethod[] = [
  {
    name: 'RSA-32768',
    enabled: true,
  },
  {
    name: 'Кузнечик',
    enabled: false,
  },
];

const meta = {
  title: 'EncryptionMethodList',
  component: EncryptionMethodList,
  parameters: {
    layout: 'padded',
  },
  args: {
    methods,
  },
} satisfies Meta<typeof EncryptionMethodList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithMethods: Story = {
  name: 'С методами',
};

export const Empty: Story = {
  name: 'Пустой',
  args: {
    methods: [],
  },
};
