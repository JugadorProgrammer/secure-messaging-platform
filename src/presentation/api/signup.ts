import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {recoveryCodesSchema} from '@/infrastructure/http/RegistrationService/recoveryCodesSchema';

const SIGNUP_PATH = '/api/signup';

export async function signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>> {
  try {
    const response = await fetch(
      SIGNUP_PATH,
      new HttpRequestBuilder().post().json(registrationRequest).build(),
    );

    return await castResponse(response, recoveryCodesSchema);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
