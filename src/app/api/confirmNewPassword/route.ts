import {cookies} from 'next/headers';
import {NextResponse} from 'next/server';

import type {NewPassword} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPassword';
import {RESTORE_PASSWORD_COOKIE} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/PasswordRecoveryService';

import {appConfig} from '@/appconfig';
import {newPasswordSchema} from '@/infrastructure/http/PasswordRecoveryService/newPasswordSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(newPasswordSchema),
  async (newPassword: NewPassword) => {
    try {
      const cookieStore = await cookies();

      if (!cookieStore.get(RESTORE_PASSWORD_COOKIE)?.value) {
        return new Response(null, {status: 400});
      }

      const result = await appConfig.resolve('passwordRecoveryService').confirmNewPassword(cookieStore.toString(), newPassword);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      const response = new NextResponse(null, {status: 204});
      response.cookies.delete(RESTORE_PASSWORD_COOKIE);

      return response;
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
