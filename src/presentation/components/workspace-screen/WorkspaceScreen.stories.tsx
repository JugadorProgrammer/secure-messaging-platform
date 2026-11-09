import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {WorkspaceScreen} from './WorkspaceScreen';

const meta = {
  title: 'WorkspaceScreen',
  component: WorkspaceScreen,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof WorkspaceScreen>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  name: 'Рабочий стол',
};
