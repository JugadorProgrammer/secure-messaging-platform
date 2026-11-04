import type {ZodType} from 'zod';

import {Result} from '@/application/Result';

export async function castResponse<T>(response: Response, schema: ZodType<T>): Promise<Result<T>> {
  if (!response.ok) {
    const error = new Error(`Ответ со статусом ${response.status}`);
    console.error(error);
    return Result.error(error);
  }

  try {
    return Result.success(schema.parse(await response.json()));
  } catch (error) {
    console.error(error);
    return Result.error(error instanceof Error ? error : new Error('Ответ не получен'));
  }
}
