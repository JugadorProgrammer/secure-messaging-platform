import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {mapEncryptionMethodList} from '@/infrastructure/http/mapEncryptionMethodList';

const ENABLED_ENCRYPTION_METHODS_PATH = '/api/encryption-methods';

export async function getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
  try {
    const response = await fetch(ENABLED_ENCRYPTION_METHODS_PATH);

    return await castResponse(response, mapEncryptionMethodList);
  } catch {
    return Result.error(new Error('Ответ не получен'));
  }
}
