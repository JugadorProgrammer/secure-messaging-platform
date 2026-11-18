import {z} from 'zod';

import type {CreateUserRequest} from '@/application/IUserService/UserService/CreateUserRequest';
import type {UserDto} from '@/application/IUserService/UserService/UserDto';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {recoveryCodesSchema} from '@/infrastructure/http/RegistrationService/recoveryCodesSchema';
import {userDtoSchema} from '@/infrastructure/http/UserService/userDtoSchema';

const USERS_PATH = '/api/users';
const USER_PATH = '/api/user';

export async function getUsers(): Promise<Result<UserDto[]>> {
  try {
    const response = await fetch(USERS_PATH, new HttpRequestBuilder().build());

    return await castResponse(response, z.array(userDtoSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}

export async function createUser(request: CreateUserRequest): Promise<Result<string[]>> {
  try {
    const response = await fetch(USER_PATH, new HttpRequestBuilder().post().json(request).build());

    return await castResponse(response, recoveryCodesSchema);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
