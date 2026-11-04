import type {DisposableCode, EmailCode, IAuthGateway, LoginCredentials, NewPassword, Registration, RestoreCode, RestorePasswordRequest} from '@/application/IAuthGateway';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {readSetCookies} from './setCookies';
import {recoveryCodesSchema} from './recoveryCodesSchema';

type HttpAuthGatewayDependencies = {
  authUrl: string;
};

const LOGIN_PATH = '/login';
const CONFIRM_EMAIL_CODE_PATH = '/confirmEmailCode';
const CONFIRM_DISPOSABLE_CODE_PATH = '/confirmDisposableCode';
const RESTORE_PASSWORD_PATH = '/restorePassword';
const CONFIRM_RESTORE_CODE_PATH = '/confirmRestoreCode';
const CONFIRM_NEW_PASSWORD_PATH = '/confirmNewPassword';
const SIGNUP_PATH = '/signup';

export class HttpAuthGateway implements IAuthGateway {
  private readonly authUrl: string;

  constructor({authUrl}: HttpAuthGatewayDependencies) {
    this.authUrl = authUrl;
  }

  login(credentials: LoginCredentials): Promise<Result<readonly string[]>> {
    return requestSetCookies(() => fetch(`${this.authUrl}${LOGIN_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        login: credentials.login,
        password: credentials.password,
      }),
    }));
  }

  confirmEmailCode(cookie: string, emailCode: EmailCode): Promise<Result<readonly string[]>> {
    return requestSetCookies(() => fetch(`${this.authUrl}${CONFIRM_EMAIL_CODE_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        cookie,
      },
      body: JSON.stringify({
        code: emailCode.code,
      }),
    }));
  }

  confirmDisposableCode(cookie: string, disposableCode: DisposableCode): Promise<Result<readonly string[]>> {
    return requestSetCookies(() => fetch(`${this.authUrl}${CONFIRM_DISPOSABLE_CODE_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        cookie,
      },
      body: JSON.stringify({
        code: disposableCode.code,
      }),
    }));
  }

  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>> {
    return requestOk(() => fetch(`${this.authUrl}${RESTORE_PASSWORD_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        email: restorePasswordRequest.email,
      }),
    }));
  }

  confirmRestoreCode(restoreCode: RestoreCode): Promise<Result<readonly string[]>> {
    return requestSetCookies(() => fetch(`${this.authUrl}${CONFIRM_RESTORE_CODE_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        code: restoreCode.code,
      }),
    }));
  }

  confirmNewPassword(cookie: string, newPassword: NewPassword): Promise<Result<readonly string[]>> {
    return requestSetCookies(() => fetch(`${this.authUrl}${CONFIRM_NEW_PASSWORD_PATH}`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        cookie,
      },
      body: JSON.stringify({
        password: newPassword.password,
      }),
    }));
  }

  async signup(registration: Registration): Promise<Result<string[]>> {
    try {
      const response = await fetch(`${this.authUrl}${SIGNUP_PATH}`, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
        },
        body: JSON.stringify(registration),
      });

      return await castResponse(response, recoveryCodesSchema);
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }
}

async function request(sendRequest: () => Promise<Response>): Promise<Result<Response>> {
  try {
    const response = await sendRequest();

    if (!response.ok) {
      const error = new Error(`Ответ со статусом ${response.status}`);
      console.error(error);
      return Result.error(error);
    }

    return Result.success(response);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}

async function requestOk(sendRequest: () => Promise<Response>): Promise<Result<void>> {
  const result = await request(sendRequest);

  if (!result.isSuccess) {
    return Result.error(result.error);
  }

  return Result.success(undefined);
}

async function requestSetCookies(sendRequest: () => Promise<Response>): Promise<Result<readonly string[]>> {
  const result = await request(sendRequest);

  if (!result.isSuccess) {
    return Result.error(result.error);
  }

  return Result.success(readSetCookies(result.success));
}
