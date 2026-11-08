'use server';

import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';

import {Result} from '@/application/Result';

import {GET as getAllEncryptionMethodsResponse} from '@/app/api/getAllEncryptionMethods/route';
import {castResponse} from '@/infrastructure/http/castResponse';
import {encryptionMethodSchema} from '@/infrastructure/http/EncryptionMethodService/encryptionMethodSchema';

export async function getAllEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
  try {
    return await castResponse(
      await getAllEncryptionMethodsResponse(),
      z.array(encryptionMethodSchema),
    );
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
