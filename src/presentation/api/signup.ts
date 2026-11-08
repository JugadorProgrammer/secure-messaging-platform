import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {recoveryCodesSchema} from '@/infrastructure/http/RegistrationService/recoveryCodesSchema';

const SIGNUP_PATH = '/api/signup';

export async function signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>> {
  try {
    const response = await fetch(
      SIGNUP_PATH,
      new HttpRequestBuilder().post().json(registrationRequest).build(),
    );

    if (!response.ok) {
      return Result.error(new Error('Не удалось зарегистрироваться.'));
    }

    const codes = recoveryCodesSchema.parse(await response.json());

    return Result.success(codes);
  } catch {
    return Result.error(new Error('Не удалось зарегистрироваться.'));
  }
}
