import type {CreateUserRequest} from '@/application/IUserService/UserService/CreateUserRequest';

import {ADMIN_ROLE} from '@/domain/role';

import {appConfig} from '@/appconfig';
import {createUserRequestSchema} from '@/infrastructure/http/UserService/createUserRequestSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  auth([ADMIN_ROLE]),
  validateHasCookies(),
  castFromJson(createUserRequestSchema),
  handleErrors(async (request: CreateUserRequest & {cookie: string}) => {
    const result = await appConfig.resolve('userService').createUser(request.cookie, request);

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  }),
);
