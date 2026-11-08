import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';
import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';
import type {ExportPdfWithSignatureRequest} from '@/application/IMessageService/MessageService/ExportPdfWithSignatureRequest';
import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import type {SignedPdfDocument} from '@/application/IMessageService/MessageService/SignedPdfDocument';
import type {Result} from '@/application/Result';

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getAllEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<MessageDto>>;
  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>>;
  getAllMessages(cookie: string): Promise<Result<MessageDto[]>>;
  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>>;
}
