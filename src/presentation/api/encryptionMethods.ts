'use server';

import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {Result} from '@/application/Result';

import {GET as getEncryptionMethods} from '@/app/api/encryption-methods/route';
import {castResponse} from '@/infrastructure/http/castResponse';
import {encryptionMethodSchema} from '@/infrastructure/http/EncryptionMethodService/encryptionMethodSchema';

export async function getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
  try {
    return await castResponse(await getEncryptionMethods(), z.array(encryptionMethodSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
