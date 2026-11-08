'use client';

import {useEffect} from 'react';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {useEncryptionMethodsStore} from '@/presentation/stores/useEncryptionMethodsStore';

type AllEncryptionMethodsPanelState = {
  encryptionMethods: EncryptionMethod[] | null;
  isEncryptionMethodsLoadFailed: boolean;
};

export function useAllEncryptionMethodsPanel(): AllEncryptionMethodsPanelState {
  const encryptionMethods = useEncryptionMethodsStore((state) => state.encryptionMethods);
  const isEncryptionMethodsLoadFailed = useEncryptionMethodsStore(
    (state) => state.isEncryptionMethodsLoadFailed,
  );
  const loadEncryptionMethods = useEncryptionMethodsStore((state) => state.loadEncryptionMethods);

  useEffect(() => {
    void loadEncryptionMethods();
  }, [loadEncryptionMethods]);

  return {
    encryptionMethods,
    isEncryptionMethodsLoadFailed,
  };
}
