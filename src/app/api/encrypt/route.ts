import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';

import {appConfig} from '@/appconfig';
import {encryptMessageRequestSchema} from '@/infrastructure/http/MessageService/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  auth(),
  validateHasCookies(),
  castFromJson(encryptMessageRequestSchema),
  handleErrors(async (request: EncryptMessageRequest & {cookie: string}) => {
    const result = await appConfig
      .resolve('messageService')
      .encryptMessage(request.cookie, request);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
