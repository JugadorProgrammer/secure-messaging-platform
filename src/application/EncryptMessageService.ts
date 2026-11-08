import type {EncryptMessageRequest, Message} from '@/domain/message';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type EncryptMessageServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class EncryptMessageService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: EncryptMessageServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<Message>> {
    return this.coordGateway.encryptMessage(cookie, request);
  }
}
