import type {ZodType} from 'zod';

import {bind} from './bind';

export function castFromJson<T extends Record<string, unknown>>(schema: ZodType<T>) {
  return async (...args: unknown[]) => {
    const [request] = args;

    if (!(request instanceof Request)) {
      return new Response(null, {status: 400});
    }

    try {
      return bind<unknown[]>(args, schema.parse(await request.json()));
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 400});
    }
  };
}
