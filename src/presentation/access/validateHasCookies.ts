import {cookies} from 'next/headers';

import {bind} from './bind';

export function validateHasCookies(emptyStatus: 400 | 401 = 401) {
  return async (...args: unknown[]) => {
    const cookie = (await cookies()).toString();

    if (cookie === '') {
      return new Response(null, {status: emptyStatus});
    }

    return bind<unknown[]>(args, {cookie});
  };
}
