import {NextResponse} from 'next/server';

import type {LoginCredentialsRequest} from '@/application/ILoginService/LoginService/LoginCredentialsRequest';

import {appConfig} from '@/appconfig';
import {loginCredentialsRequestSchema} from '@/infrastructure/http/LoginService/loginCredentialsRequestSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(loginCredentialsRequestSchema),
  handleErrors(async (loginCredentialsRequest: LoginCredentialsRequest) => {
    const result = await appConfig.resolve('loginService').login(loginCredentialsRequest);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    const response = new NextResponse(null, {status: 204});
    writeSetCookies(response, result.success);

    return response;
  }),
);
