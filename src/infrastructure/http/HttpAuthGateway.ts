import type {DisposableCode, EmailCode, IAuthGateway, LoginCredentials} from '@/application/IAuthGateway';
import {Result} from '@/application/Result';

import {readSetCookies} from './setCookies';

type HttpAuthGatewayDependencies = {
  authUrl: string;
};

const LOGIN_PATH = '/login';
const CONFIRM_EMAIL_CODE_PATH = '/confirmEmailCode';
const CONFIRM_DISPOSABLE_CODE_PATH = '/confirmDisposableCode';

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
}

async function requestSetCookies(sendRequest: () => Promise<Response>): Promise<Result<readonly string[]>> {
  try {
    const response = await sendRequest();

    if (!response.ok) {
      const error = new Error(`Ответ со статусом ${response.status}`);
      console.error(error);
      return Result.error(error);
    }

    return Result.success(readSetCookies(response));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
