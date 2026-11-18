'use client';

import type {ChangeEvent, FormEvent} from 'react';

import {Notification} from '@/presentation/components/notification/Notification';

import {useCreateUserForm} from './useCreateUserForm';

import styles from './CreateUserForm.module.css';

export function CreateUserForm() {
  const {
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
  } = useCreateUserForm();

  function handleLoginChange(event: ChangeEvent<HTMLInputElement>): void {
    setLogin(event.target.value);
  }

  function handleEmailChange(event: ChangeEvent<HTMLInputElement>): void {
    setEmail(event.target.value);
  }

  function handlePasswordChange(event: ChangeEvent<HTMLInputElement>): void {
    setPassword(event.target.value);
  }

  function handleRoleChange(event: ChangeEvent<HTMLSelectElement>): void {
    const value = Number(event.target.value);

    if (Number.isInteger(value)) {
      setRole(value);
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await submit();
  }

  const hasRoles = roles !== null && roles.length > 0 && role !== null;

  if (recoveryCodes !== null) {
    return (
      <div className={styles['create-user-form']}>
        <h2 className={styles['create-user-form__title']}>Создание пользователя</h2>
        <p className={styles['create-user-form__hint']}>
          Пользователь создан. Сохраните коды восстановления. Они показываются один раз.
        </p>
        <ul className={styles['create-user-form__codes']}>
          {recoveryCodes.map((code) => (
            <li key={code} className={styles['create-user-form__code']}>
              {code}
            </li>
          ))}
        </ul>
        <button className={styles['create-user-form__submit']} type="button" onClick={resetForm}>
          Создать ещё
        </button>
      </div>
    );
  }

  return (
    <form className={styles['create-user-form']} onSubmit={handleSubmit}>
      <h2 className={styles['create-user-form__title']}>Создание пользователя</h2>
      <label className={styles['create-user-form__field']}>
        <span className={styles['create-user-form__label']}>Логин</span>
        <input
          className={styles['create-user-form__input']}
          name="login"
          autoComplete="username"
          value={login}
          onChange={handleLoginChange}
          required
        />
        <p className={styles['create-user-form__hint']}>
          От 3 до 32 символов, латиница и цифры, начинается с буквы
        </p>
      </label>
      <label className={styles['create-user-form__field']}>
        <span className={styles['create-user-form__label']}>Почта</span>
        <input
          className={styles['create-user-form__input']}
          name="email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={handleEmailChange}
          required
        />
      </label>
      <label className={styles['create-user-form__field']}>
        <span className={styles['create-user-form__label']}>Пароль</span>
        <input
          className={styles['create-user-form__input']}
          name="password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={handlePasswordChange}
          required
        />
        <p className={styles['create-user-form__hint']}>
          От 8 до 64 символов, строчная и заглавная буква и цифра
        </p>
      </label>
      <label className={styles['create-user-form__field']}>
        <span className={styles['create-user-form__label']}>Роль пользователя</span>
        <select
          className={styles['create-user-form__input']}
          name="role"
          value={role ?? ''}
          onChange={handleRoleChange}
          disabled={!hasRoles}
          required
        >
          {hasRoles ? (
            roles.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))
          ) : (
            <option value="">
              {roles === null && !isRolesFailed ? 'Загрузка…' : 'Нет доступных ролей'}
            </option>
          )}
        </select>
      </label>
      <button className={styles['create-user-form__submit']} type="submit" disabled={isSubmitting || !hasRoles}>
        Создать
      </button>
      {isRolesFailed ? <Notification message="Не удалось загрузить список ролей." /> : null}
      {notice !== null ? (
        <Notification key={notice.notificationDate} message={notice.message} />
      ) : null}
    </form>
  );
}
