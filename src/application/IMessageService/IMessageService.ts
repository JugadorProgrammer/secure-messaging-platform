import type {Message} from '@/domain/message';

import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';
import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';
import type {ExportPdfWithSignatureRequest} from '@/application/IMessageService/MessageService/ExportPdfWithSignatureRequest';
import type {SignedPdfDocument} from '@/application/IMessageService/MessageService/SignedPdfDocument';
import type {Result} from '@/application/Result';

export interface IMessageService {
  getAllMessages(cookie: string): Promise<Result<Message[]>>;
  encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<Message>>;
  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>>;
  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>>;
}
