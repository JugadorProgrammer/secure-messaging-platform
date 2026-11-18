'use client';

import {create} from 'zustand';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import {getUsers} from '@/presentation/api/users';

type UsersState = {
  users: UserDto[] | null;
  isUsersLoadFailed: boolean;
  loadUsers: () => Promise<void>;
  reloadUsers: () => Promise<void>;
};

const RETRY_LIMIT = 3;
const RETRY_DELAY_MS = 5000;

let loadPromise: Promise<void> | null = null;

export const useUsersStore = create<UsersState>((set, get) => ({
  users: null,
  isUsersLoadFailed: false,
  async loadUsers() {
    if (get().users !== null) {
      return;
    }

    if (loadPromise !== null) {
      return loadPromise;
    }

    loadPromise = (async () => {
      for (let retry = 0; retry <= RETRY_LIMIT; retry += 1) {
        const result = await getUsers();

        if (result.isSuccess) {
          set({
            users: result.success,
            isUsersLoadFailed: false,
          });
          return;
        }

        if (retry === RETRY_LIMIT) {
          set({
            users: null,
            isUsersLoadFailed: true,
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
  async reloadUsers() {
    const result = await getUsers();

    if (result.isSuccess) {
      set({
        users: result.success,
        isUsersLoadFailed: false,
      });
    }
  },
}));

export function resetUsersStore(): void {
  loadPromise = null;
  useUsersStore.setState({
    users: null,
    isUsersLoadFailed: false,
  });
}
