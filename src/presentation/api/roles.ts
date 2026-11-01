'use server';

import {z} from 'zod';

import type {Role} from '@/domain/role';

import {Result} from '@/application/Result';

import {GET as getRolesResponse} from '@/app/api/roles/route';
import {castResponse} from '@/infrastructure/http/castResponse';
import {roleSchema} from '@/infrastructure/http/roleSchema';

export async function getRoles(): Promise<Result<Role[]>> {
  try {
    return await castResponse(await getRolesResponse(), z.array(roleSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
