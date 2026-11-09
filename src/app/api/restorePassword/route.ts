import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';

import {appConfig} from '@/appconfig';
import {restorePasswordRequestSchema} from '@/infrastructure/http/PasswordRecoveryService/restorePasswordRequestSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(restorePasswordRequestSchema),
  handleErrors(async (restorePasswordRequest: RestorePasswordRequest) => {
    const result = await appConfig
      .resolve('passwordRecoveryService')
      .restorePassword(restorePasswordRequest);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return new Response(null, {status: 204});
  }),
);
