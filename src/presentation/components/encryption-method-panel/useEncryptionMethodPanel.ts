'use client';

import {useEffect, useState} from 'react';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';

type EncryptionMethodPanelState = {
  methods: EncryptionMethod[] | null;
  isFailed: boolean;
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

export function useEncryptionMethodPanel(): EncryptionMethodPanelState {
  const [state, setState] = useState<EncryptionMethodPanelState>({
    methods: null,
    isFailed: false,
  });

  useEffect(() => {
    let isCancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    function load(retry: number): void {
      void getEnabledEncryptionMethods().then((result) => {
        if (isCancelled) {
          return;
        }

        if (result.isSuccess) {
          setState({methods: result.success, isFailed: false});
          return;
        }

        if (retry === RETRY_LIMIT) {
          setState({methods: null, isFailed: true});
          return;
        }

        timer = setTimeout(() => {
          load(retry + 1);
        }, RETRY_DELAY_MS);
      });
    }

    load(0);

    return () => {
      isCancelled = true;
      clearTimeout(timer);
    };
  }, []);

  return state;
}
