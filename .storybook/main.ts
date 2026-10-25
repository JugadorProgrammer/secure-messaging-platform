import type {StorybookConfig} from '@storybook/nextjs-vite';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  addons: [],
  framework: '@storybook/nextjs-vite',
  core: {
    disableTelemetry: true,
  },
};

export default config;
