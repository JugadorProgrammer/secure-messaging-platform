import {Result} from '@/application/Result';

const LOGIN_PATH = '/api/login';
const CONFIRM_EMAIL_CODE_PATH = '/api/confirmEmailCode';

type LoginRequest = {
  login: string;
  password: string;
};

export async function login(credentials: LoginRequest): Promise<Result<void>> {
  try {
    const response = await fetch(LOGIN_PATH, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        login: credentials.login,
        password: credentials.password,
      }),
    });

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
    const response = await fetch(CONFIRM_EMAIL_CODE_PATH, {
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
