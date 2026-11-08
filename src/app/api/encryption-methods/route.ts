import {appConfig} from '@/appconfig';

import {compose} from '@/presentation/access/auth';
import {handleErrors} from '@/presentation/access/handleErrors';

export const GET = compose(
  handleErrors(async () => {
    const result = await appConfig.resolve('encryptionMethodService').getEnabledEncryptionMethods();

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
