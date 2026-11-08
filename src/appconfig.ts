import 'server-only';

import {asClass, asFunction, createContainer, InjectionMode} from 'awilix';

import {EncryptMessageService} from '@/application/EncryptMessageService';
import {DecryptMessageService} from '@/application/DecryptMessageService';
import {EncryptionMethodService} from '@/application/EncryptionMethodService';
import {LoginService} from '@/application/LoginService';
import {PasswordRecoveryService} from '@/application/PasswordRecoveryService';
import {RegistrationService} from '@/application/RegistrationService';
import {RoleService} from '@/application/RoleService';
import {HttpAuthGateway} from '@/infrastructure/http/HttpAuthGateway';
import {HttpCoordGateway} from '@/infrastructure/http/HttpCoordGateway';

export const appConfig = createContainer({
  injectionMode: InjectionMode.PROXY,
}).register({
  coordUrl: asFunction(readCoordUrl).singleton(),
  authUrl: asFunction(readAuthUrl).singleton(),
  coordGateway: asClass(HttpCoordGateway).singleton(),
  authGateway: asClass(HttpAuthGateway).singleton(),
  roleService: asClass(RoleService).singleton(),
  encryptionMethodService: asClass(EncryptionMethodService).singleton(),
  encryptMessageService: asClass(EncryptMessageService).singleton(),
  decryptMessageService: asClass(DecryptMessageService).singleton(),
  loginService: asClass(LoginService).singleton(),
  passwordRecoveryService: asClass(PasswordRecoveryService).singleton(),
  registrationService: asClass(RegistrationService).singleton(),
});

function readCoordUrl(): string {
  return readEnv('COORD_URL');
}

function readAuthUrl(): string {
  return readEnv('AUTH_URL');
}

function readEnv(name: string): string {
  const value = process.env[name];

  if (value === undefined || value.trim() === '') {
    throw new Error(`Не задана переменная ${name}`);
  }

  return value;
}
