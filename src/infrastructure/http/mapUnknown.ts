import {Result} from '@/application/Result';

type FieldReader<T> = (value: unknown, name: string) => Result<T>;

export function mapString(value: unknown, name: string): Result<string> {
  if (typeof value !== 'string' || value.trim() === '') {
    return Result.error(new Error(`Поле ${name} должно быть непустой строкой`));
  }

  return Result.success(value);
}

export function mapList<T>(
  value: unknown,
  readItem: (item: unknown) => Result<T>,
): Result<T[]> {
  if (!Array.isArray(value)) {
    return Result.error(new Error('Ожидался список'));
  }

  const items: T[] = [];

  for (const item of value) {
    const mapped = readItem(item);

    if (!mapped.isSuccess) {
      return mapped;
    }

    items.push(mapped.success);
  }

  return Result.success(items);
}

export function mapRecord<T extends object>(
  value: unknown,
  fields: {[K in keyof T]: FieldReader<T[K]>},
): Result<T> {
  if (!isRecord(value)) {
    return Result.error(new Error('Ожидался объект'));
  }

  const mapped: Partial<T> = {};

  for (const name of Object.keys(fields) as (keyof T & string)[]) {
    const field = fields[name](value[name], name);

    if (!field.isSuccess) {
      return field;
    }

    mapped[name] = field.success;
  }

  return Result.success(mapped as T);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
