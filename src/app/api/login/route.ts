import {NextResponse} from 'next/server';

import type {LoginCredentials} from '@/application/IAuthGateway';

import {appConfig} from '@/appconfig';
import {loginCredentialsSchema} from '@/infrastructure/http/loginCredentialsSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(loginCredentialsSchema),
  async (credentials: LoginCredentials) => {
    try {
      const result = await appConfig.resolve('loginService').login(credentials);

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
