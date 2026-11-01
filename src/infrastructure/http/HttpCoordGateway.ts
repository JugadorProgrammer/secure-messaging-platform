import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {ICoordGateway, LoginCredentials} from '@/application/ICoordGateway';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {encryptionMethodSchema} from './encryptionMethodSchema';
import {roleSchema} from './roleSchema';
import {sessionTokenSchema} from './sessionTokenSchema';

type HttpCoordGatewayDependencies = {
  coordUrl: string;
};

const ROLES_PATH = '/api/roles';
const ENABLED_ENCRYPTION_METHODS_PATH = '/getEnabledEncryptionMethods';
const LOGIN_PATH = '/login';

export class HttpCoordGateway implements ICoordGateway {
  private readonly coordUrl: string;

  constructor({coordUrl}: HttpCoordGatewayDependencies) {
    this.coordUrl = coordUrl;
  }

  async getRoles(): Promise<Result<Role[]>> {
    try {
      const response = await fetch(`${this.coordUrl}${ROLES_PATH}`);

      return await castResponse(response, z.array(roleSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    try {
      const response = await fetch(`${this.coordUrl}${ENABLED_ENCRYPTION_METHODS_PATH}`);

      return await castResponse(response, z.array(encryptionMethodSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async login(credentials: LoginCredentials): Promise<Result<{token: string}>> {
    try {
      const response = await fetch(`${this.coordUrl}${LOGIN_PATH}`, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
        },
        body: JSON.stringify({
          login: credentials.login,
          password: credentials.password,
        }),
      });

      return await castResponse(response, sessionTokenSchema);
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }
}
