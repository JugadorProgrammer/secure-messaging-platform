import {cookies} from 'next/headers';
import {NextResponse} from 'next/server';

import type {EmailCode} from '@/application/ILoginService/LoginService/EmailCode';

import {appConfig} from '@/appconfig';
import {emailCodeSchema} from '@/infrastructure/http/LoginService/emailCodeSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(emailCodeSchema),
  async (emailCode: EmailCode) => {
    try {
      const cookie = (await cookies()).toString();

      if (cookie === '') {
        return new Response(null, {status: 400});
      }

      const result = await appConfig.resolve('loginService').confirmEmailCode(cookie, emailCode);

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
