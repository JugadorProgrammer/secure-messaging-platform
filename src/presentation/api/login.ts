import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';

const LOGIN_PATH = '/api/login';
const CONFIRM_EMAIL_CODE_PATH = '/api/confirmEmailCode';
const CONFIRM_DISPOSABLE_CODE_PATH = '/api/confirmDisposableCode';

type LoginRequest = {
  login: string;
  password: string;
};

export async function login(credentials: LoginRequest): Promise<Result<void>> {
  try {
    const response = await fetch(
      LOGIN_PATH,
      new HttpRequestBuilder()
        .post()
        .json({
          login: credentials.login,
          password: credentials.password,
        })
        .build(),
    );

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось войти.'));
  } catch {
    return Result.error(new Error('Не удалось войти.'));
  }
}

export async function confirmEmailCode(code: string): Promise<Result<void>> {
  try {
    const response = await fetch(
      CONFIRM_EMAIL_CODE_PATH,
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

export async function confirmDisposableCode(code: string): Promise<Result<void>> {
  try {
    const response = await fetch(
      CONFIRM_DISPOSABLE_CODE_PATH,
      new HttpRequestBuilder().post().json({code}).build(),
    );

    if (response.ok) {
      return Result.success(undefined);
    }

    return Result.error(new Error('Не удалось подтвердить код восстановления.'));
  } catch {
    return Result.error(new Error('Не удалось подтвердить код восстановления.'));
  }
}
