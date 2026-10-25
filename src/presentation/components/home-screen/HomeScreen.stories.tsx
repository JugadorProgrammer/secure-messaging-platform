import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {HomeScreen} from './HomeScreen';

const meta = {
  title: 'HomeScreen',
  component: HomeScreen,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof HomeScreen>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  name: 'Главный экран',
};
