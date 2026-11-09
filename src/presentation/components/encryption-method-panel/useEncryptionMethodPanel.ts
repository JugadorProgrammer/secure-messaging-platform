'use client';

import {useEffect} from 'react';
import {useShallow} from 'zustand/react/shallow';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {
  selectEnabledEncryptionMethods,
  useEncryptionMethodsStore,
} from '@/presentation/stores/useEncryptionMethodsStore';

type EncryptionMethodPanelState = {
  encryptionMethods: EncryptionMethod[] | null;
  isEncryptionMethodsLoadFailed: boolean;
};

export function useEncryptionMethodPanel(): EncryptionMethodPanelState {
  const encryptionMethods = useEncryptionMethodsStore(useShallow(selectEnabledEncryptionMethods));
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
