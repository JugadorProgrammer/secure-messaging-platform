import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {SetEncryptionMethodAvailabilityRequest} from '@/application/IEncryptionMethodService/EncryptionMethodService/SetEncryptionMethodAvailabilityRequest';
import type {Result} from '@/application/Result';

export interface IEncryptionMethodService {
  getAllEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  setEncryptionMethodAvailability(
    cookie: string,
    request: SetEncryptionMethodAvailabilityRequest,
  ): Promise<Result<void>>;
}
