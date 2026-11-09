'use client';

import {useEffect, useState} from 'react';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import {getUsers} from '@/presentation/api/users';

type UserListPanelState = {
  users: UserDto[] | null;
  isUsersLoadFailed: boolean;
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

export function useUserListPanel(): UserListPanelState {
  const [users, setUsers] = useState<UserDto[] | null>(null);
  const [isUsersLoadFailed, setIsUsersLoadFailed] = useState(false);

  useEffect(() => {
    let isCancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    async function load(): Promise<void> {
      for (let retry = 0; retry <= RETRY_LIMIT; retry += 1) {
        if (isCancelled) {
          return;
        }

        const result = await getUsers();

        if (isCancelled) {
          return;
        }

        if (result.isSuccess) {
          setUsers(result.success);
          setIsUsersLoadFailed(false);
          return;
        }

        if (retry === RETRY_LIMIT) {
          setUsers(null);
          setIsUsersLoadFailed(true);
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

  return {
    users,
    isUsersLoadFailed,
  };
}
