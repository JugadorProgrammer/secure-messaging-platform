import type {Preview} from '@storybook/nextjs-vite';
import {mocked, sb} from 'storybook/test';

import {Result} from '@/application/Result';
import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';
import {getAllMessages} from '@/presentation/api/messages';
import {getRoles} from '@/presentation/api/roles';

import '../src/app/globals.css';

import {sampleEncryptionMethods} from './sampleEncryptionMethods';
import {sampleRoles} from './sampleRoles';

sb.mock(import('../src/presentation/api/encrypt.ts'));
sb.mock(import('../src/presentation/api/decrypt.ts'));
sb.mock(import('../src/presentation/api/encryptionMethods.ts'));
sb.mock(import('../src/presentation/api/exportPdfWithSignature.ts'));
sb.mock(import('../src/presentation/api/login.ts'));
sb.mock(import('../src/presentation/api/messages.ts'));
sb.mock(import('../src/presentation/api/passwordRecovery.ts'));
sb.mock(import('../src/presentation/api/roles.ts'));
sb.mock(import('../src/presentation/api/signup.ts'));

const preview: Preview = {
  beforeEach() {
    mocked(getRoles).mockResolvedValue(Result.success(sampleRoles));
    mocked(getEnabledEncryptionMethods).mockResolvedValue(Result.success(sampleEncryptionMethods));
    mocked(getAllMessages).mockResolvedValue(Result.success([]));
  },
};

export default preview;
