import type {DecryptMessageRequest} from '@/application/IMessageService/MessageService/DecryptMessageRequest';
import type {DecryptMessageResult} from '@/application/IMessageService/MessageService/DecryptMessageResult';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {decryptMessageResultSchema} from '@/infrastructure/http/MessageService/messageSchema';

const DECRYPT_PATH = '/api/decrypt';

export async function decryptMessage(
  request: DecryptMessageRequest,
): Promise<Result<DecryptMessageResult>> {
  try {
    const response = await fetch(
      DECRYPT_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    return await castResponse(response, decryptMessageResultSchema);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
