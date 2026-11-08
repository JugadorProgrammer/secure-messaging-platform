import type {Message} from '@/domain/message';
import type {EncryptMessageRequest} from '@/application/EncryptMessageRequest';

import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {messageSchema} from '@/infrastructure/http/messageSchema';

const ENCRYPT_PATH = '/api/encrypt';

export async function encryptMessage(request: EncryptMessageRequest): Promise<Result<Message>> {
  try {
    const response = await fetch(
      ENCRYPT_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    if (!response.ok) {
      return Result.error(new Error('Не удалось зашифровать сообщение.'));
    }

    return Result.success(messageSchema.parse(await response.json()));
  } catch {
    return Result.error(new Error('Не удалось зашифровать сообщение.'));
  }
}
