import type {EncryptMessageRequest} from '@/application/IMessageService/MessageService/EncryptMessageRequest';
import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {messageDtoSchema} from '@/infrastructure/http/MessageService/messageSchema';

const ENCRYPT_PATH = '/api/encrypt';

export async function encryptMessage(request: EncryptMessageRequest): Promise<Result<MessageDto>> {
  try {
    const response = await fetch(
      ENCRYPT_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    return await castResponse(response, messageDtoSchema);
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
