import {Result} from '@/application/Result';

export async function castResponse<T>(
  response: Response,
  map: (body: unknown) => Result<T>,
): Promise<Result<T>> {
  if (!response.ok) {
    return Result.error(new Error(`Ответ со статусом ${response.status}`));
  }

  let body: unknown;

  try {
    body = await response.json();
  } catch {
    return Result.error(new Error('Ответ не получен'));
  }

  return map(body);
}
