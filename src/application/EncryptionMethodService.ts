import type {EncryptionMethod} from '@/domain/encryptionMethod';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type EncryptionMethodServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class EncryptionMethodService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: EncryptionMethodServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    return this.coordGateway.getEnabledEncryptionMethods();
  }
}
