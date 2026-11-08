import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';

import {appConfig} from '@/appconfig';
import {restorePasswordRequestSchema} from '@/infrastructure/http/PasswordRecoveryService/restorePasswordRequestSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(restorePasswordRequestSchema),
  async (restorePasswordRequest: RestorePasswordRequest) => {
    try {
      const result = await appConfig.resolve('passwordRecoveryService').restorePassword(restorePasswordRequest);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      return new Response(null, {status: 204});
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
