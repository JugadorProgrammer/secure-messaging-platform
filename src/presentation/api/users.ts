import {z} from 'zod';

import type {UserDto} from '@/application/IUserService/UserService/UserDto';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {userDtoSchema} from '@/infrastructure/http/UserService/userDtoSchema';

const USERS_PATH = '/api/users';

export async function getUsers(): Promise<Result<UserDto[]>> {
  try {
    const response = await fetch(USERS_PATH, new HttpRequestBuilder().build());

    return await castResponse(response, z.array(userDtoSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
