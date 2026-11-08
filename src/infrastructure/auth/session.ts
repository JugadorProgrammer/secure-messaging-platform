import 'server-only';

import {cookies} from 'next/headers';

import type {User} from '@/domain/user';

import {sessionClaimsSchema} from './sessionClaimsSchema';

export const SESSION_COOKIE = 'session';

export type SessionUser = User & {
  roles: readonly string[];
};

function decodeJwtPayload(token: string): unknown {
  const parts = token.split('.');

  if (parts.length !== 3) {
    throw new Error('Некорректный JWT');
  }

  return JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf8'));
}

export async function getUser(): Promise<SessionUser | null> {
  const session = (await cookies()).get(SESSION_COOKIE);

  if (!session?.value) {
    return null;
  }

  try {
    return sessionClaimsSchema.parse(decodeJwtPayload(session.value));
  } catch (error) {
    console.error(error);
    return null;
  }
}
