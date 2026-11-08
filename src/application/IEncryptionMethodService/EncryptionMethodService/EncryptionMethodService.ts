import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {ICoordGateway} from '@/application/ICoordGateway/ICoordGateway';
import type {IEncryptionMethodService} from '@/application/IEncryptionMethodService/IEncryptionMethodService';
import type {Result} from '@/application/Result';

type EncryptionMethodServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class EncryptionMethodService implements IEncryptionMethodService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: EncryptionMethodServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    return this.coordGateway.getEnabledEncryptionMethods();
  }
}
