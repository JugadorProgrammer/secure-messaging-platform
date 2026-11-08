import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {AdminScreen} from './AdminScreen';

const meta = {
  title: 'AdminScreen',
  component: AdminScreen,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof AdminScreen>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  name: 'По умолчанию',
};
