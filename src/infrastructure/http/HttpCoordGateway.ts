import {z} from 'zod';

import type {EncryptionMethod} from '@/domain/encryptionMethod';
import type {Role} from '@/domain/role';

import type {ICoordGateway} from '@/application/ICoordGateway/ICoordGateway';
import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';
import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';
import type {ExportPdfWithSignatureRequest} from '@/application/IMessageService/MessageService/ExportPdfWithSignatureRequest';
import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import type {SignedPdfDocument} from '@/application/IMessageService/MessageService/SignedPdfDocument';
import {Result} from '@/application/Result';

import {castResponse} from './castResponse';
import {encryptionMethodSchema} from './EncryptionMethodService/encryptionMethodSchema';
import {HttpRequestBuilder} from './HttpRequestBuilder';
import {decryptMessageResultSchema, messageDtoSchema} from './MessageService/messageSchema';
import {roleSchema} from './RoleService/roleSchema';

type HttpCoordGatewayDependencies = {
  coordUrl: string;
};

const ROLES_PATH = '/api/roles';
const ENABLED_ENCRYPTION_METHODS_PATH = '/api/getEnabledEncryptionMethods';
const ENCRYPT_PATH = '/api/encrypt';
const DECRYPT_PATH = '/api/decrypt';
const GET_ALL_MESSAGES_PATH = '/api/getAllMessages';
const EXPORT_PDF_WITH_SIGNATURE_PATH = '/api/exportPdfWithSignature';
const DEFAULT_PDF_FILENAME = 'message.pdf';

export class HttpCoordGateway implements ICoordGateway {
  private readonly coordUrl: string;

  constructor({coordUrl}: HttpCoordGatewayDependencies) {
    this.coordUrl = coordUrl;
  }

  async getRoles(): Promise<Result<Role[]>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${ROLES_PATH}`,
        new HttpRequestBuilder().build(),
      );

      return await castResponse(response, z.array(roleSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async getEnabledEncryptionMethods(): Promise<Result<EncryptionMethod[]>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${ENABLED_ENCRYPTION_METHODS_PATH}`,
        new HttpRequestBuilder().build(),
      );

      return await castResponse(response, z.array(encryptionMethodSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async encryptMessage(cookie: string, request: EncryptMessageRequest): Promise<Result<MessageDto>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${ENCRYPT_PATH}`,
        new HttpRequestBuilder().post().cookie(cookie).json(request).build(),
      );

      return await castResponse(response, messageDtoSchema);
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async decryptMessage(
    cookie: string,
    request: DecryptMessageRequest,
  ): Promise<Result<DecryptMessageResult>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${DECRYPT_PATH}`,
        new HttpRequestBuilder().post().cookie(cookie).json(request).build(),
      );

      return await castResponse(response, decryptMessageResultSchema);
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async getAllMessages(cookie: string): Promise<Result<MessageDto[]>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${GET_ALL_MESSAGES_PATH}`,
        new HttpRequestBuilder().cookie(cookie).build(),
      );

      return await castResponse(response, z.array(messageDtoSchema));
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }

  async exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>> {
    try {
      const response = await fetch(
        `${this.coordUrl}${EXPORT_PDF_WITH_SIGNATURE_PATH}`,
        new HttpRequestBuilder().post().cookie(cookie).json(request).build(),
      );

      if (!response.ok) {
        return Result.error(new Error('Ответ не получен'));
      }

      const content = await response.arrayBuffer();

      if (content.byteLength === 0) {
        return Result.error(new Error('Ответ не получен'));
      }

      return Result.success({
        content,
        filename: readFilename(response.headers.get('content-disposition')) ?? DEFAULT_PDF_FILENAME,
      });
    } catch (error) {
      console.error(error);
      return Result.error(new Error('Ответ не получен'));
    }
  }
}

function readFilename(header: string | null): string | null {
  if (header === null) {
    return null;
  }

  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(header);

  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return utf8Match[1];
    }
  }

  const basicMatch = /filename="?([^";]+)"?/i.exec(header);

  return basicMatch?.[1] ?? null;
}
