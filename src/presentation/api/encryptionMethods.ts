import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {SetEncryptionMethodAvailabilityRequest} from '@/application/IEncryptionMethodService/EncryptionMethodService/SetEncryptionMethodAvailabilityRequest';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {encryptionMethodSchema} from '@/infrastructure/http/EncryptionMethodService/encryptionMethodSchema';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';

const ALL_ENCRYPTION_METHODS_PATH = '/api/getAllEncryptionMethods';
const SET_ENCRYPTION_METHOD_AVAILABILITY_PATH = '/api/setEncryptionMethodAvailability';

export async function getAllEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
  try {
    const response = await fetch(ALL_ENCRYPTION_METHODS_PATH, new HttpRequestBuilder().build());

    return await castResponse(response, z.array(encryptionMethodSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}

export async function setEncryptionMethodAvailability(
  request: SetEncryptionMethodAvailabilityRequest,
): Promise<Result<void>> {
  try {
    const response = await fetch(
      SET_ENCRYPTION_METHOD_AVAILABILITY_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    if (response.ok) {
      return Result.success(undefined);
    }

    const error = new Error(`Ответ со статусом ${response.status}`);
    console.error(error);
    return Result.error(error);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
