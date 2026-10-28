import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {Result} from '@/application/Result';

import {mapBoolean, mapList, mapRecord, mapString} from './mapUnknown';

export function mapEncryptionMethodList(body: unknown): Result<EncryptionMethod[]> {
  return mapList(body, mapEncryptionMethod);
}

function mapEncryptionMethod(value: unknown): Result<EncryptionMethod> {
  return mapRecord(value, {
    name: mapString,
    enabled: mapBoolean,
  });
}
