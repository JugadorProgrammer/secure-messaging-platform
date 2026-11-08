import {appConfig} from '@/appconfig';

import {auth, compose} from '@/presentation/access/auth';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const GET = compose(auth(), validateHasCookies(), async (request: {cookie: string}) => {
  try {
    const result = await appConfig.resolve('messageService').getAllMessages(request.cookie);

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
