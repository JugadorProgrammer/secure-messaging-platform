import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {
  DecryptMessageRequest,
  DecryptMessageResult,
  EncryptMessageRequest,
  ExportPdfWithSignatureRequest,
  Message,
  SignedPdfDocument,
} from '@/domain/message';
import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

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
