import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';

import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {decryptMessageResultSchema} from '@/infrastructure/http/messageSchema';

const DECRYPT_PATH = '/api/decrypt';

export async function decryptMessage(
  request: DecryptMessageRequest,
): Promise<Result<DecryptMessageResult>> {
  try {
    const response = await fetch(
      DECRYPT_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    if (!response.ok) {
      return Result.error(new Error('Не удалось расшифровать сообщение.'));
    }

    return Result.success(decryptMessageResultSchema.parse(await response.json()));
  } catch {
    return Result.error(new Error('Не удалось расшифровать сообщение.'));
  }
}
