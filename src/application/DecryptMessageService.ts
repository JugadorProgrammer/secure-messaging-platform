import type {DecryptMessageRequest, DecryptMessageResult} from '@/domain/message';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type DecryptMessageServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class DecryptMessageService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: DecryptMessageServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>> {
    return this.coordGateway.decryptMessage(cookie, request);
  }
}
