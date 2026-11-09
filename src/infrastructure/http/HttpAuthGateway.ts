import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {DisposableCodeRequest} from '@/application/ILoginService/LoginService/DisposableCodeRequest';
import type {EmailCodeRequest} from '@/application/ILoginService/LoginService/EmailCodeRequest';
import type {LoginCredentialsRequest} from '@/application/ILoginService/LoginService/LoginCredentialsRequest';
import type {NewPasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPasswordRequest';
import type {RestoreCodeRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCodeRequest';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {HttpRequestBuilder} from './HttpRequestBuilder';
import {readSetCookies} from './setCookies';
import {recoveryCodesSchema} from './RegistrationService/recoveryCodesSchema';

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

  login(loginCredentialsRequest: LoginCredentialsRequest): Promise<Result<readonly string[]>> {
    return requestSetCookies(() =>
      fetch(
        `${this.authUrl}${LOGIN_PATH}`,
        new HttpRequestBuilder()
          .post()
          .json({
            login: loginCredentialsRequest.login,
            password: loginCredentialsRequest.password,
          })
          .build(),
      ),
    );
  }

  confirmEmailCode(
    cookie: string,
    emailCodeRequest: EmailCodeRequest,
  ): Promise<Result<readonly string[]>> {
    return requestSetCookies(() =>
      fetch(
        `${this.authUrl}${CONFIRM_EMAIL_CODE_PATH}`,
        new HttpRequestBuilder()
          .post()
          .cookie(cookie)
          .json({
            code: emailCodeRequest.code,
          })
          .build(),
      ),
    );
  }

  confirmDisposableCode(
    cookie: string,
    disposableCodeRequest: DisposableCodeRequest,
  ): Promise<Result<readonly string[]>> {
    return requestSetCookies(() =>
      fetch(
        `${this.authUrl}${CONFIRM_DISPOSABLE_CODE_PATH}`,
        new HttpRequestBuilder()
          .post()
          .cookie(cookie)
          .json({
            code: disposableCodeRequest.code,
          })
          .build(),
      ),
    );
  }

  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>> {
    return requestOk(() =>
      fetch(
        `${this.authUrl}${RESTORE_PASSWORD_PATH}`,
        new HttpRequestBuilder()
          .post()
          .json({
            email: restorePasswordRequest.email,
          })
          .build(),
      ),
    );
  }

  confirmRestoreCode(restoreCodeRequest: RestoreCodeRequest): Promise<Result<readonly string[]>> {
    return requestSetCookies(() =>
      fetch(
        `${this.authUrl}${CONFIRM_RESTORE_CODE_PATH}`,
        new HttpRequestBuilder()
          .post()
          .json({
            code: restoreCodeRequest.code,
          })
          .build(),
      ),
    );
  }

  confirmNewPassword(
    cookie: string,
    newPasswordRequest: NewPasswordRequest,
  ): Promise<Result<readonly string[]>> {
    return requestSetCookies(() =>
      fetch(
        `${this.authUrl}${CONFIRM_NEW_PASSWORD_PATH}`,
        new HttpRequestBuilder()
          .post()
          .cookie(cookie)
          .json({
            password: newPasswordRequest.password,
          })
          .build(),
      ),
    );
  }

  async signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>> {
    try {
      const response = await fetch(
        `${this.authUrl}${SIGNUP_PATH}`,
        new HttpRequestBuilder().post().json(registrationRequest).build(),
      );

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
