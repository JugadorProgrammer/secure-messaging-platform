import type {Message} from '@/domain/message';

import type {ICoordGateway} from '@/application/ICoordGateway/ICoordGateway';
import type {IMessageService} from '@/application/IMessageService/IMessageService';
import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';
import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';
import type {ExportPdfWithSignatureRequest} from '@/application/IMessageService/MessageService/ExportPdfWithSignatureRequest';
import type {SignedPdfDocument} from '@/application/IMessageService/MessageService/SignedPdfDocument';
import type {Result} from '@/application/Result';

type MessageServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class MessageService implements IMessageService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: MessageServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getAllMessages(cookie: string): Promise<Result<Message[]>> {
    return this.coordGateway.getAllMessages(cookie);
  }

  encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<Message>> {
    return this.coordGateway.encryptMessage(cookie, request);
  }

  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>> {
    return this.coordGateway.decryptMessage(cookie, request);
  }

  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>> {
    return this.coordGateway.exportPdfWithSignature(cookie, request);
  }
}
