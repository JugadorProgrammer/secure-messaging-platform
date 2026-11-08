import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Message} from '@/domain/message';
import type {Role} from '@/domain/role';

import type {DecryptMessageRequest} from '@/application/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/DecryptMessageResult';
import type {EncryptMessageRequest} from '@/application/EncryptMessageRequest';
import type {ExportPdfWithSignatureRequest} from '@/application/ExportPdfWithSignatureRequest';
import type {Result} from '@/application/Result';
import type {SignedPdfDocument} from '@/application/SignedPdfDocument';

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<Message>>;
  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>>;
  getAllMessages(cookie: string): Promise<Result<Message[]>>;
  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>>;
}
