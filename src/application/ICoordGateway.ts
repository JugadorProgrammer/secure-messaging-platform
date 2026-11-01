import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

export type LoginCredentials = {
  login: string;
  password: string;
};

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  login(credentials: LoginCredentials): Promise<Result<{token: string}>>;
}
