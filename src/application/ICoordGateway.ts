import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Message} from '@/domain/message';
import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  getAllMessages(cookie: string): Promise<Result<Message[]>>;
}
