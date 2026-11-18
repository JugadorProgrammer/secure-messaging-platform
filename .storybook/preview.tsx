import type {Preview} from '@storybook/nextjs-vite';
import {mocked, sb} from 'storybook/test';

import {Result} from '@/application/Result';
import {
  getAllEncryptionMethods,
  setEncryptionMethodAvailability,
} from '@/presentation/api/encryptionMethods';
import {getAllMessages} from '@/presentation/api/messages';
import {getRoles} from '@/presentation/api/roles';
import {createUser, getUsers} from '@/presentation/api/users';
import {resetEncryptionMethodsStore} from '@/presentation/stores/useEncryptionMethodsStore';
import {resetUsersStore} from '@/presentation/stores/useUsersStore';

import '../src/app/globals.css';

import {sampleEncryptionMethods} from './sampleEncryptionMethods';
import {sampleRoles} from './sampleRoles';
import {sampleUsers} from './sampleUsers';

sb.mock(import('../src/presentation/api/encrypt.ts'));
sb.mock(import('../src/presentation/api/decrypt.ts'));
sb.mock(import('../src/presentation/api/encryptionMethods.ts'));
sb.mock(import('../src/presentation/api/exportPdfWithSignature.ts'));
sb.mock(import('../src/presentation/api/login.ts'));
sb.mock(import('../src/presentation/api/messages.ts'));
sb.mock(import('../src/presentation/api/passwordRecovery.ts'));
sb.mock(import('../src/presentation/api/roles.ts'));
sb.mock(import('../src/presentation/api/signup.ts'));
sb.mock(import('../src/presentation/api/users.ts'));

const preview: Preview = {
  beforeEach() {
    resetEncryptionMethodsStore();
    resetUsersStore();
    mocked(getRoles).mockResolvedValue(Result.success(sampleRoles));
    mocked(getAllEncryptionMethods).mockResolvedValue(Result.success(sampleEncryptionMethods));
    mocked(setEncryptionMethodAvailability).mockResolvedValue(Result.success(undefined));
    mocked(getAllMessages).mockResolvedValue(Result.success([]));
    mocked(getUsers).mockResolvedValue(Result.success(sampleUsers));
    mocked(createUser).mockResolvedValue(Result.success(['AB12CD34', 'EF56GH78']));
  },
};

export default preview;
