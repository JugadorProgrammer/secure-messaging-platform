import {z} from 'zod';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import {Result} from '@/application/Result';

import {castResponse} from '@/infrastructure/http/castResponse';
import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';
import {messageDtoSchema} from '@/infrastructure/http/MessageService/messageSchema';

const MESSAGES_PATH = '/api/messages';

export async function getAllMessages(): Promise<Result<MessageDto[]>> {
  try {
    const response = await fetch(MESSAGES_PATH, new HttpRequestBuilder().build());

    return await castResponse(response, z.array(messageDtoSchema));
  } catch (error) {
    console.error(error);
    return Result.error(new Error('Ответ не получен'));
  }
}
