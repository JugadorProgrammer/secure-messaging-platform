import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {
  DecryptMessageRequest,
  DecryptMessageResult,
  ExportPdfWithSignatureRequest,
  Message,
  SignedPdfDocument,
} from '@/domain/message';
import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

export interface ICoordGateway {
  getRoles(): Promise<Result<Role[]>>;
  getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>>;
  getAllMessages(cookie: string): Promise<Result<Message[]>>;
  decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>>;
  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>>;
}
