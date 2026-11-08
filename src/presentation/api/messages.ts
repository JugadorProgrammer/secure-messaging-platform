import {z} from 'zod';

import type {Message} from '@/domain/message';

import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {messageSchema} from '@/infrastructure/http/messageSchema';

const MESSAGES_PATH = '/api/messages';

export async function getAllMessages(): Promise<Result<Message[]>> {
  try {
    const response = await fetch(MESSAGES_PATH, new HttpRequestBuilder().build());

    if (!response.ok) {
      return Result.error(new Error('Не удалось загрузить журнал сообщений.'));
    }

    return Result.success(z.array(messageSchema).parse(await response.json()));
  } catch {
    return Result.error(new Error('Не удалось загрузить журнал сообщений.'));
  }
}
