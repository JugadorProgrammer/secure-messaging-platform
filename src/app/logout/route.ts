import {cookies} from 'next/headers';
import {redirect} from 'next/navigation';

import {SESSION_COOKIE} from '@/infrastructure/auth/session';

import {auth, compose} from '@/presentation/access/auth';

export const GET = compose(auth(), async function GET() {
  (await cookies()).delete(SESSION_COOKIE);
  redirect('/login');
});
