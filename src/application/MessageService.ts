import type {Message} from '@/domain/message';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type MessageServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class MessageService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: MessageServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getAllMessages(cookie: string): Promise<Result<Message[]>> {
    return this.coordGateway.getAllMessages(cookie);
  }
}
