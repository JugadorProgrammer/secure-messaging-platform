'use client';

import {useEffect, useState} from 'react';
import type {Dispatch, SetStateAction} from 'react';

import type {Role} from '@/domain/role';

import {
  emailPattern,
  loginPattern,
  passwordPattern,
} from '@/infrastructure/http/RegistrationService/registrationRequestSchema';
import {getRoles} from '@/presentation/api/roles';
import {createUser} from '@/presentation/api/users';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import {useUsersStore} from '@/presentation/stores/useUsersStore';

type CreateUserFormState = {
  login: string;
  setLogin: Dispatch<SetStateAction<string>>;
  email: string;
  setEmail: Dispatch<SetStateAction<string>>;
  password: string;
  setPassword: Dispatch<SetStateAction<string>>;
  role: string;
  setRole: Dispatch<SetStateAction<string>>;
  roles: Role[] | null;
  isRolesFailed: boolean;
  recoveryCodes: readonly string[] | null;
  isSubmitting: boolean;
  notice: ScreenNotice | null;
  submit: () => Promise<void>;
  resetForm: () => void;
};

export function useCreateUserForm(): CreateUserFormState {
  const [login, setLogin] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState('');
  const [roles, setRoles] = useState<Role[] | null>(null);
  const [isRolesFailed, setIsRolesFailed] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<readonly string[] | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();
  const reloadUsers = useUsersStore((state) => state.reloadUsers);

  useEffect(() => {
    let isCancelled = false;

    void getRoles().then((result) => {
      if (isCancelled) {
        return;
      }

      if (!result.isSuccess) {
        setRoles(null);
        setIsRolesFailed(true);
        return;
      }

      setRoles(result.success);
      setIsRolesFailed(false);
      setRole((current) => {
        if (current !== '' && result.success.some((item) => item.name === current)) {
          return current;
        }

        return result.success[0]?.name ?? '';
      });
    });

    return () => {
      isCancelled = true;
    };
  }, []);

  function validateForm(): string | null {
    if (!loginPattern.test(login)) {
      return 'Логин: от 3 до 32 символов, латиница и цифры, начинается с буквы.';
    }

    if (!emailPattern.test(email)) {
      return 'Почта указана неверно.';
    }

    if (!passwordPattern.test(password)) {
      return 'Пароль: от 8 до 64 символов, строчная и заглавная буква и цифра.';
    }

    if (role.trim() === '') {
      return 'Нет доступных ролей.';
    }

    return null;
  }

  async function submit(): Promise<void> {
    const error = validateForm();

    if (error !== null) {
      show(error);
      return;
    }

    setIsSubmitting(true);
    dismiss();

    const result = await createUser({login, email, password, role});

    setPassword('');
    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    setRecoveryCodes(result.success);
    await reloadUsers();
  }

  function resetForm(): void {
    setLogin('');
    setEmail('');
    setPassword('');
    setRole(roles?.[0]?.name ?? '');
    setRecoveryCodes(null);
    dismiss();
  }

  return {
    login,
    setLogin,
    email,
    setEmail,
    password,
    setPassword,
    role,
    setRole,
    roles,
    isRolesFailed,
    recoveryCodes,
    isSubmitting,
    notice,
    submit,
    resetForm,
  };
}
