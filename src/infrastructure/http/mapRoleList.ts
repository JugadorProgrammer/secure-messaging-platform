import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

import {mapList, mapRecord, mapString} from './mapUnknown';

export function mapRoleList(body: unknown): Result<Role[]> {
  return mapList(body, mapRole);
}

function mapRole(value: unknown): Result<Role> {
  return mapRecord(value, {
    id: mapString,
    name: mapString,
    description: mapString,
  });
}
