import type {SetEncryptionMethodAvailabilityRequest} from '@/application/IEncryptionMethodService/EncryptionMethodService/SetEncryptionMethodAvailabilityRequest';

import {ADMIN_ROLE} from '@/domain/role';

import {appConfig} from '@/appconfig';
import {setEncryptionMethodAvailabilityRequestSchema} from '@/infrastructure/http/EncryptionMethodService/setEncryptionMethodAvailabilityRequestSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  auth([ADMIN_ROLE]),
  validateHasCookies(),
  castFromJson(setEncryptionMethodAvailabilityRequestSchema),
  handleErrors(async (request: SetEncryptionMethodAvailabilityRequest & {cookie: string}) => {
    const result = await appConfig
      .resolve('encryptionMethodService')
      .setEncryptionMethodAvailability(request.cookie, request);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return new Response(null, {status: 200});
  }),
);
