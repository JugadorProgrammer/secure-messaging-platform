import {NextResponse} from 'next/server';

import type {DisposableCodeRequest} from '@/application/ILoginService/LoginService/DisposableCodeRequest';

import {appConfig} from '@/appconfig';
import {disposableCodeRequestSchema} from '@/infrastructure/http/LoginService/disposableCodeRequestSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  castFromJson(disposableCodeRequestSchema),
  validateHasCookies(400),
  handleErrors(async (request: DisposableCodeRequest & {cookie: string}) => {
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
  }),
);
