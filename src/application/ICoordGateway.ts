import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
}
