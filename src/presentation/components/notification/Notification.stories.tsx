import type {Meta, StoryObj} from '@storybook/nextjs-vite';

import {Notification} from './Notification';

const meta = {
  title: 'Notification',
  component: Notification,
  parameters: {
    layout: 'fullscreen',
  },
  args: {
    message: 'Не удалось загрузить список ролей.',
  },
} satisfies Meta<typeof Notification>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Failure: Story = {
  name: 'Ошибка загрузки ролей',
};
