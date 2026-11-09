import {NextResponse} from 'next/server';

import type {RestoreCodeRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCodeRequest';
import {RESTORE_PASSWORD_COOKIE} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/PasswordRecoveryService';

import {appConfig} from '@/appconfig';
import {restoreCodeRequestSchema} from '@/infrastructure/http/PasswordRecoveryService/restoreCodeRequestSchema';
import {writeSetCookies} from '@/infrastructure/http/setCookies';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(restoreCodeRequestSchema),
  handleErrors(async (restoreCodeRequest: RestoreCodeRequest) => {
    const result = await appConfig
      .resolve('passwordRecoveryService')
      .confirmRestoreCode(restoreCodeRequest);

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
  }),
);
