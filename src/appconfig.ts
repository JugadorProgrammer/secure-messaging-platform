import 'server-only';

import {asClass, asFunction, createContainer, InjectionMode} from 'awilix';

import {RoleService} from '@/application/RoleService';
import {HttpCoordGateway} from '@/infrastructure/http/HttpCoordGateway';

export const appConfig = createContainer({
  injectionMode: InjectionMode.PROXY,
}).register({
  coordUrl: asFunction(readCoordUrl).singleton(),
  coordGateway: asClass(HttpCoordGateway).singleton(),
  roleService: asClass(RoleService).singleton(),
});

function readCoordUrl(): string {
  return readEnv('COORD_URL');
}

function readEnv(name: string): string {
  const value = process.env[name];

  if (value === undefined || value.trim() === '') {
    throw new Error(`Не задана переменная ${name}`);
  }

  return value;
}
