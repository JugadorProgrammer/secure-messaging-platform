import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';

import {appConfig} from '@/appconfig';
import {decryptMessageRequestSchema} from '@/infrastructure/http/MessageService/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  auth(),
  validateHasCookies(),
  castFromJson(decryptMessageRequestSchema),
  handleErrors(async (request: DecryptMessageRequest & {cookie: string}) => {
    const result = await appConfig
      .resolve('messageService')
      .decryptMessage(request.cookie, request);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
