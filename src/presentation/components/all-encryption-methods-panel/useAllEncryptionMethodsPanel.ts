'use client';

import {useEffect} from 'react';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import {useEncryptionMethodsStore} from '@/presentation/stores/useEncryptionMethodsStore';

type AllEncryptionMethodsPanelState = {
  encryptionMethods: EncryptionMethod[] | null;
  isEncryptionMethodsLoadFailed: boolean;
  pendingAvailabilityMethodName: string | null;
  notice: ReturnType<typeof useScreenNotice>['notice'];
  setAvailability: (method: EncryptionMethod, enabled: boolean) => Promise<void>;
};

export function useAllEncryptionMethodsPanel(): AllEncryptionMethodsPanelState {
  const encryptionMethods = useEncryptionMethodsStore((state) => state.encryptionMethods);
  const isEncryptionMethodsLoadFailed = useEncryptionMethodsStore(
    (state) => state.isEncryptionMethodsLoadFailed,
  );
  const pendingAvailabilityMethodName = useEncryptionMethodsStore(
    (state) => state.pendingAvailabilityMethodName,
  );
  const loadEncryptionMethods = useEncryptionMethodsStore((state) => state.loadEncryptionMethods);
  const setEncryptionMethodAvailability = useEncryptionMethodsStore(
    (state) => state.setEncryptionMethodAvailability,
  );
  const {notice, show} = useScreenNotice();

  useEffect(() => {
    void loadEncryptionMethods();
  }, [loadEncryptionMethods]);

  async function setAvailability(method: EncryptionMethod, enabled: boolean): Promise<void> {
    const result = await setEncryptionMethodAvailability({name: method.name, enabled});

    if (!result.isSuccess) {
      show(result.error.message);
    }
  }

  return {
    encryptionMethods,
    isEncryptionMethodsLoadFailed,
    pendingAvailabilityMethodName,
    notice,
    setAvailability,
  };
}
