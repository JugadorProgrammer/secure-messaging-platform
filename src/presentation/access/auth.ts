import {redirect} from 'next/navigation';

import {getUser} from '@/infrastructure/auth/session';

import {bind} from './bind';

export {compose} from './compose';

function isApi(args: readonly unknown[]): boolean {
  const [request] = args;

  if (!(request instanceof Request)) {
    return false;
  }

  return new URL(request.url).pathname.startsWith('/api/');
}

function reject(args: readonly unknown[], status: 401 | 403): Response {
  if (isApi(args)) {
    return new Response(null, {status});
  }

  redirect(status === 401 ? '/login' : '/');
}

export function auth(allowedRoles?: readonly string[]) {
  return async (...args: unknown[]) => {
    const user = await getUser();

    if (!user) {
      return reject(args, 401);
    }

    if (allowedRoles?.length && !allowedRoles.some((role) => user.roles.includes(role))) {
      return reject(args, 403);
    }

    return bind<unknown[]>(args, {user, roles: user.roles});
  };
}

export async function noAuth(...args: unknown[]) {
  if (await getUser()) {
    return reject(args, 403);
  }

  return args;
}

