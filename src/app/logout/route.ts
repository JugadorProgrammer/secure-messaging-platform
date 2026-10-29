import {cookies} from 'next/headers';
import {redirect} from 'next/navigation';

const SESSION_COOKIE = 'session';

export async function GET() {
  (await cookies()).delete(SESSION_COOKIE);
  redirect('/');
}
