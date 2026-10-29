import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {Header} from './Header';

const meta = {
  title: 'Header',
  component: Header,
  parameters: {
    layout: 'fullscreen',
  },
  args: {
    isAuthorized: false,
  },
} satisfies Meta<typeof Header>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Guest: Story = {
  name: 'Гость',
};

export const Authorized: Story = {
  name: 'Авторизован',
  args: {
    isAuthorized: true,
  },
};
