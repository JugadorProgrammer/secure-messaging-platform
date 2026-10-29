import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {ICoordGateway} from '@/application/ICoordGateway';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {mapEncryptionMethodList} from './mapEncryptionMethodList';
import {mapRoleList} from './mapRoleList';

type HttpCoordGatewayDependencies = {
  coordUrl: string;
};

const ROLES_PATH = '/api/roles';
const ENABLED_ENCRYPTION_METHODS_PATH = '/getEnabledEncryptionMethods';

export class HttpCoordGateway implements ICoordGateway {
  private readonly coordUrl: string;

  constructor({coordUrl}: HttpCoordGatewayDependencies) {
    this.coordUrl = coordUrl;
  }

  async getRoles(): Promise<Result<Role[]>> {
    try {
      const response = await fetch(`${this.coordUrl}${ROLES_PATH}`);

      return await castResponse(response, mapRoleList);
    } catch {
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    try {
      const response = await fetch(`${this.coordUrl}${ENABLED_ENCRYPTION_METHODS_PATH}`);

      return await castResponse(response, mapEncryptionMethodList);
    } catch {
      return Result.error(new Error('Ответ не получен'));
    }
  }
}
