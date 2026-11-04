import type {Registration} from '@/application/IAuthGateway';
import {Result} from '@/application/Result';

import {recoveryCodesSchema} from '@/infrastructure/http/recoveryCodesSchema';

const SIGNUP_PATH = '/api/signup';

export async function signup(registration: Registration): Promise<Result<string[]>> {
  try {
    const response = await fetch(SIGNUP_PATH, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify(registration),
    });

    if (!response.ok) {
      return Result.error(new Error('Не удалось зарегистрироваться.'));
    }

    const codes = recoveryCodesSchema.parse(await response.json());

    return Result.success(codes);
  } catch {
    return Result.error(new Error('Не удалось зарегистрироваться.'));
  }
}
