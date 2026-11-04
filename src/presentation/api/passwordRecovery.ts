import {Result} from '@/application/Result';

const RESTORE_PASSWORD_PATH = '/api/restorePassword';
const CONFIRM_RESTORE_CODE_PATH = '/api/confirmRestoreCode';
const CONFIRM_NEW_PASSWORD_PATH = '/api/confirmNewPassword';

export async function restorePassword(email: string): Promise<Result<void>> {
  try {
    const response = await fetch(RESTORE_PASSWORD_PATH, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        email,
      }),
    });

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось отправить письмо.'));
  } catch {
    return Result.error(new Error('Не удалось отправить письмо.'));
  }
}

export async function confirmRestoreCode(code: string): Promise<Result<void>> {
  try {
    const response = await fetch(CONFIRM_RESTORE_CODE_PATH, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        code,
      }),
    });

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось подтвердить код.'));
  } catch {
    return Result.error(new Error('Не удалось подтвердить код.'));
  }
}

export async function confirmNewPassword(password: string): Promise<Result<void>> {
  try {
    const response = await fetch(CONFIRM_NEW_PASSWORD_PATH, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        password,
      }),
    });

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось сменить пароль.'));
  } catch {
    return Result.error(new Error('Не удалось сменить пароль.'));
  }
}
