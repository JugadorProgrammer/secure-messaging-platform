import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';

import {appConfig} from '@/appconfig';
import {registrationRequestSchema} from '@/infrastructure/http/RegistrationService/registrationRequestSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(registrationRequestSchema),
  handleErrors(async (registrationRequest: RegistrationRequest) => {
    const result = await appConfig.resolve('registrationService').signup(registrationRequest);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
