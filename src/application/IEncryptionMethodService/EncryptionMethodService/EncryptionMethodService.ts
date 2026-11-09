import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {ICoordGateway} from '@/application/ICoordGateway/ICoordGateway';
import type {IEncryptionMethodService} from '@/application/IEncryptionMethodService/IEncryptionMethodService';
import type {SetEncryptionMethodAvailabilityRequest} from '@/application/IEncryptionMethodService/EncryptionMethodService/SetEncryptionMethodAvailabilityRequest';
import type {Result} from '@/application/Result';

type EncryptionMethodServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class EncryptionMethodService implements IEncryptionMethodService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: EncryptionMethodServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getAllEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    return this.coordGateway.getAllEncryptionMethods();
  }

  setEncryptionMethodAvailability(
    cookie: string,
    request: SetEncryptionMethodAvailabilityRequest,
  ): Promise<Result<void>> {
    return this.coordGateway.setEncryptionMethodAvailability(cookie, request);
  }
}
