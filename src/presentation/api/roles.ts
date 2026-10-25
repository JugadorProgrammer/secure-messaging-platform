import type {Role} from '@/domain/role';

import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {mapRoleList} from '@/infrastructure/http/mapRoleList';

const ROLES_PATH = '/api/roles';

export async function getRoles(): Promise<Result<Role[]>> {
  try {
    const response = await fetch(ROLES_PATH);

    return await castResponse(response, mapRoleList);
  } catch {
    return Result.error(new Error('Ответ не получен'));
  }
}
