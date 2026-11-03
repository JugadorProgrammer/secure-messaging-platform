import 'server-only';

import {cookies} from 'next/headers';

export const SESSION_COOKIE = 'session';

export type User = {
  roles: readonly string[];
};

export async function getUser(): Promise<User | null> {
  const session = (await cookies()).get(SESSION_COOKIE);

  if (!session?.value) {
    return null;
  }

  return {
    roles: [],
  };
}
