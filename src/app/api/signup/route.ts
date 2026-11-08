import type {Registration} from '@/application/IRegistrationService/RegistrationService/Registration';

import {appConfig} from '@/appconfig';
import {signupSchema} from '@/infrastructure/http/RegistrationService/signupSchema';

import {compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  castFromJson(signupSchema),
  async (registration: Registration) => {
    try {
      const result = await appConfig.resolve('registrationService').signup(registration);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      return Response.json(result.success);
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
