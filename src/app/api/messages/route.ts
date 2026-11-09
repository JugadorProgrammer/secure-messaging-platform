import {appConfig} from '@/appconfig';

import {auth, compose} from '@/presentation/access/auth';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const GET = compose(
  auth(),
  validateHasCookies(),
  handleErrors(async (request: {cookie: string}) => {
    const result = await appConfig.resolve('messageService').getAllMessages(request.cookie);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
