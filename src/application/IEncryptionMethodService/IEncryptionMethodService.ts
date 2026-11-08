import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {Result} from '@/application/Result';

export interface IEncryptionMethodService {
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
}
