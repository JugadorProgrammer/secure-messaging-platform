import {cookies} from 'next/headers';

import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';

import {appConfig} from '@/appconfig';
import {encryptMessageRequestSchema} from '@/infrastructure/http/MessageService/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  auth(),
  castFromJson(encryptMessageRequestSchema),
  async (request: EncryptMessageRequest) => {
    try {
      const cookie = (await cookies()).toString();

      if (cookie === '') {
        return new Response(null, {status: 401});
      }

      const result = await appConfig.resolve('messageService').encryptMessage(cookie, request);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      return Response.json(result.success);
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
