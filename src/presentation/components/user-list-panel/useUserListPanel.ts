'use client';

import {useEffect} from 'react';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';

import {useUsersStore} from '@/presentation/stores/useUsersStore';

type UserListPanelState = {
  users: UserDto[] | null;
  isUsersLoadFailed: boolean;
};

export function useUserListPanel(): UserListPanelState {
  const users = useUsersStore((state) => state.users);
  const isUsersLoadFailed = useUsersStore((state) => state.isUsersLoadFailed);
  const loadUsers = useUsersStore((state) => state.loadUsers);

  useEffect(() => {
    void loadUsers();
  }, [loadUsers]);

  return {
    users,
    isUsersLoadFailed,
  };
}
