import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {NavigationPanel} from './NavigationPanel';

const meta = {
  title: 'NavigationPanel',
  component: NavigationPanel,
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof NavigationPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  name: 'Кнопки',
};
