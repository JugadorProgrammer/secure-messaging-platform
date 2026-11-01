import {NextResponse} from 'next/server';

import type {LoginCredentials} from '@/application/ICoordGateway';

import {appConfig} from '@/appconfig';
import {SESSION_COOKIE} from '@/infrastructure/auth/session';
import {loginCredentialsSchema} from '@/infrastructure/http/loginCredentialsSchema';

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

      response.cookies.set(SESSION_COOKIE, result.success.token, {
        httpOnly: true,
        sameSite: 'lax',
        path: '/',
      });

      return response;
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
