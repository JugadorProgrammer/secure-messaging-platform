'use client';

import {create} from 'zustand';

import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {MessageEncryptionMethod} from '@/domain/message';
import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

import type {SetEncryptionMethodAvailabilityRequest} from '@/application/IEncryptionMethodService/EncryptionMethodService/SetEncryptionMethodAvailabilityRequest';
import {Result} from '@/application/Result';

import {
  getAllEncryptionMethods,
  setEncryptionMethodAvailability as requestSetEncryptionMethodAvailability,
} from '@/presentation/api/encryptionMethods';

type EncryptionMethodsState = {
  encryptionMethods: EncryptionMethod[] | null;
  isEncryptionMethodsLoadFailed: boolean;
  pendingAvailabilityMethodName: string | null;
  loadEncryptionMethods: () => Promise<void>;
  setEncryptionMethodAvailability: (
    request: SetEncryptionMethodAvailabilityRequest,
  ) => Promise<Result<void>>;
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
  pendingAvailabilityMethodName: null,
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
  async setEncryptionMethodAvailability(request) {
    if (get().pendingAvailabilityMethodName !== null) {
      return Result.error(new Error('Дождитесь завершения предыдущего переключения.'));
    }

    set({pendingAvailabilityMethodName: request.name});

    const result = await requestSetEncryptionMethodAvailability(request);

    if (!result.isSuccess) {
      set({pendingAvailabilityMethodName: null});
      return result;
    }

    const encryptionMethods = get().encryptionMethods;

    set({
      pendingAvailabilityMethodName: null,
      encryptionMethods:
        encryptionMethods === null
          ? null
          : encryptionMethods.map((method) =>
              method.name === request.name ? {...method, enabled: request.enabled} : method,
            ),
    });

    return result;
  },
}));

export function resetEncryptionMethodsStore(): void {
  loadPromise = null;
  useEncryptionMethodsStore.setState({
    encryptionMethods: null,
    isEncryptionMethodsLoadFailed: false,
    pendingAvailabilityMethodName: null,
  });
}
