'use client';

import {create} from 'zustand';

import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {MessageEncryptionMethod} from '@/domain/message';
import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

import {getAllEncryptionMethods} from '@/presentation/api/encryptionMethods';

type EncryptionMethodsState = {
  encryptionMethods: EncryptionMethod[] | null;
  isEncryptionMethodsLoadFailed: boolean;
  loadEncryptionMethods: () => Promise<void>;
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

let loadPromise: Promise<void> | null = null;

export function selectEnabledEncryptionMethods(
  state: EncryptionMethodsState,
): EncryptionMethod[] | null {
  if (state.encryptionMethods === null) {
    return null;
  }

  return state.encryptionMethods.filter((method) => method.enabled);
}

export function selectEnabledMessageEncryptionMethods(
  state: EncryptionMethodsState,
): MessageEncryptionMethod[] | null {
  const enabledEncryptionMethods = selectEnabledEncryptionMethods(state);

  if (enabledEncryptionMethods === null) {
    return null;
  }

  return enabledEncryptionMethods.map((item) => item.name).filter(isMessageEncryptionMethod);
}

function isMessageEncryptionMethod(value: string): value is MessageEncryptionMethod {
  return (MESSAGE_ENCRYPTION_METHODS as readonly string[]).includes(value);
}

export const useEncryptionMethodsStore = create<EncryptionMethodsState>((set, get) => ({
  encryptionMethods: null,
  isEncryptionMethodsLoadFailed: false,
  async loadEncryptionMethods() {
    if (get().encryptionMethods !== null) {
      return;
    }

    if (loadPromise !== null) {
      return loadPromise;
    }

    loadPromise = (async () => {
      for (let retry = 0; retry <= RETRY_LIMIT; retry += 1) {
        const result = await getAllEncryptionMethods();

        if (result.isSuccess) {
          set({
            encryptionMethods: result.success,
            isEncryptionMethodsLoadFailed: false,
          });
          return;
        }

        if (retry === RETRY_LIMIT) {
          set({
            encryptionMethods: null,
            isEncryptionMethodsLoadFailed: true,
          });
          return;
        }

        await new Promise((resolve) => {
          setTimeout(resolve, RETRY_DELAY_MS);
        });
      }
    })().finally(() => {
      loadPromise = null;
    });

    return loadPromise;
  },
}));

export function resetEncryptionMethodsStore(): void {
  loadPromise = null;
  useEncryptionMethodsStore.setState({
    encryptionMethods: null,
    isEncryptionMethodsLoadFailed: false,
  });
}
