import {cookies} from 'next/headers';

import {appConfig} from '@/appconfig';

import {auth, compose} from '@/presentation/access/auth';

export const GET = compose(auth(), async function GET() {
  try {
    const cookie = (await cookies()).toString();

    if (cookie === '') {
      return new Response(null, {status: 401});
    }

    const result = await appConfig.resolve('messageService').getAllMessages(cookie);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  } catch (error) {
    console.error(error);
    return new Response(null, {status: 503});
  }
});
