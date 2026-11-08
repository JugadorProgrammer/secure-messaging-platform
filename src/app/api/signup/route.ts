import type {Registration} from '@/application/IRegistrationService/RegistrationService/Registration';

import {appConfig} from '@/appconfig';
import {signupSchema} from '@/infrastructure/http/RegistrationService/signupSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';

export const POST = compose(
  castFromJson(signupSchema),
  handleErrors(async (registration: Registration) => {
    const result = await appConfig.resolve('registrationService').signup(registration);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
