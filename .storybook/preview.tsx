import type {Preview} from '@storybook/nextjs-vite';
import {mocked, sb} from 'storybook/test';

import {Result} from '@/application/Result';
import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';
import {getRoles} from '@/presentation/api/roles';

import '../src/app/globals.css';

import {sampleEncryptionMethods} from './sampleEncryptionMethods';
import {sampleRoles} from './sampleRoles';

sb.mock(import('../src/presentation/api/encryptionMethods.ts'));
sb.mock(import('../src/presentation/api/roles.ts'));

const preview: Preview = {
  beforeEach() {
    mocked(getRoles).mockResolvedValue(Result.success(sampleRoles));
    mocked(getEnabledEncryptionMethods).mockResolvedValue(Result.success(sampleEncryptionMethods));
  },
};

export default preview;
