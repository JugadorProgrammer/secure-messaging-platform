import {NextResponse} from 'next/server';

import type {RestoreCode} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCode';
import {RESTORE_PASSWORD_COOKIE} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/PasswordRecoveryService';

import {appConfig} from '@/appconfig';
import {restoreCodeSchema} from '@/infrastructure/http/PasswordRecoveryService/restoreCodeSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(restoreCodeSchema),
  async (restoreCode: RestoreCode) => {
    try {
      const result = await appConfig.resolve('passwordRecoveryService').confirmRestoreCode(restoreCode);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      const response = new NextResponse(null, {status: 204});
      writeSetCookies(response, result.success);
      response.cookies.set(RESTORE_PASSWORD_COOKIE, '1', {
        httpOnly: true,
        path: '/',
        sameSite: 'lax',
      });

      return response;
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
