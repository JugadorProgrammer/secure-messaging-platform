'use client';

import {useEffect, useState} from 'react';

import type {Role} from '@/domain/role';

import {getRoles} from '@/presentation/api/roles';

type RoleListLoaderState = {
  roles: Role[] | null;
  isFailed: boolean;
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

export function useRoleListLoader(): RoleListLoaderState {
  const [state, setState] = useState<RoleListLoaderState>({
    roles: null,
    isFailed: false,
  });

  useEffect(() => {
    let isCancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    async function load(): Promise<void> {
      for (let retry = 0; retry <= RETRY_LIMIT; retry += 1) {
        if (isCancelled) {
          return;
        }

        const result = await getRoles();

        if (isCancelled) {
          return;
        }

        if (result.isSuccess) {
          setState({roles: result.success, isFailed: false});
          return;
        }

        if (retry === RETRY_LIMIT) {
          setState({roles: null, isFailed: true});
          return;
        }

        await new Promise((resolve) => {
          timer = setTimeout(resolve, RETRY_DELAY_MS);
        });
      }
    }

    void load();

    return () => {
      isCancelled = true;
      clearTimeout(timer);
    };
  }, []);

  return state;
}
