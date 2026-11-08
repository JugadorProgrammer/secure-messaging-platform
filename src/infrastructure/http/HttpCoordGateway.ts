import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Message} from '@/domain/message';
import type {Role} from '@/domain/role';

import type {ICoordGateway} from '@/application/ICoordGateway';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {encryptionMethodSchema} from './encryptionMethodSchema';
import {HttpRequestBuilder} from './HttpRequestBuilder';
import {messageSchema} from './messageSchema';
import {roleSchema} from './roleSchema';

type HttpCoordGatewayDependencies = {
  coordUrl: string;
};

const ROLES_PATH = '/api/roles';
const ENABLED_ENCRYPTION_METHODS_PATH = '/getEnabledEncryptionMethods';
const GET_ALL_MESSAGES_PATH = '/getAllMessages';

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

  async getAllMessages(cookie: string): Promise<Result<Message[]>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${GET_ALL_MESSAGES_PATH}`,
        new HttpRequestBuilder().cookie(cookie).build(),
      );

      return await castResponse(response, z.array(messageSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }
}
