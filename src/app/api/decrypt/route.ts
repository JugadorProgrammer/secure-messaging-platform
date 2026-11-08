import {cookies} from 'next/headers';

import type {DecryptMessageRequest} from '@/domain/message';

import {appConfig} from '@/appconfig';
import {decryptMessageRequestSchema} from '@/infrastructure/http/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  auth(),
  castFromJson(decryptMessageRequestSchema),
  async (request: DecryptMessageRequest) => {
    try {
      const cookie = (await cookies()).toString();

      if (cookie === '') {
        return new Response(null, {status: 401});
      }

      const result = await appConfig.resolve('decryptMessageService').decryptMessage(cookie, request);

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
