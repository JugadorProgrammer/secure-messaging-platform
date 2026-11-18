import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';

const RESTORE_PASSWORD_PATH = '/api/restorePassword';
const CONFIRM_RESTORE_CODE_PATH = '/api/confirmRestoreCode';
const CONFIRM_NEW_PASSWORD_PATH = '/api/confirmNewPassword';

export async function restorePassword(email: string): Promise<Result<void>> {
  try {
    const response = await fetch(
      RESTORE_PASSWORD_PATH,
      new HttpRequestBuilder().post().json({email}).build(),
    );

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
    const response = await fetch(
      CONFIRM_RESTORE_CODE_PATH,
      new HttpRequestBuilder().post().json({code}).build(),
    );

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
    const response = await fetch(
      CONFIRM_NEW_PASSWORD_PATH,
      new HttpRequestBuilder().post().json({password}).build(),
    );

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось сменить пароль.'));
  } catch {
    return Result.error(new Error('Не удалось сменить пароль.'));
  }
}
