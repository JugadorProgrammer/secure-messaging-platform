import {z} from 'zod';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {messageDtoSchema} from '@/infrastructure/http/MessageService/messageSchema';

const MESSAGES_PATH = '/api/messages';

export async function getAllMessages(): Promise<Result<MessageDto[]>> {
  try {
    const response = await fetch(MESSAGES_PATH, new HttpRequestBuilder().build());

    if (!response.ok) {
      return Result.error(new Error('Не удалось загрузить журнал сообщений.'));
    }

    return Result.success(z.array(messageDtoSchema).parse(await response.json()));
  } catch {
    return Result.error(new Error('Не удалось загрузить журнал сообщений.'));
  }
}
