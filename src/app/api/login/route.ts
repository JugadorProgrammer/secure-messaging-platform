import {NextResponse} from 'next/server';

import type {LoginCredentials} from '@/application/ILoginService/LoginService/LoginCredentials';

import {appConfig} from '@/appconfig';
import {loginCredentialsSchema} from '@/infrastructure/http/LoginService/loginCredentialsSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(loginCredentialsSchema),
  handleErrors(async (credentials: LoginCredentials) => {
    const result = await appConfig.resolve('loginService').login(credentials);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    const response = new NextResponse(null, {status: 204});
    writeSetCookies(response, result.success);

    return response;
  }),
);
