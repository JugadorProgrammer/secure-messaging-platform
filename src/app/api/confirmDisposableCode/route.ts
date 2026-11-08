import {NextResponse} from 'next/server';

import type {DisposableCode} from '@/application/ILoginService/LoginService/DisposableCode';

import {appConfig} from '@/appconfig';
import {disposableCodeSchema} from '@/infrastructure/http/LoginService/disposableCodeSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  castFromJson(disposableCodeSchema),
  validateHasCookies(400),
  async (request: DisposableCode & {cookie: string}) => {
    try {
      const result = await appConfig
        .resolve('loginService')
        .confirmDisposableCode(request.cookie, request);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      const response = new NextResponse(null, {status: 204});
      writeSetCookies(response, result.success);

      return response;
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
