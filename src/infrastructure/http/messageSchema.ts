import {z} from 'zod';

import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

export const messageEncryptionMethodSchema = z.enum(MESSAGE_ENCRYPTION_METHODS);

const nonEmptyString = z.string().refine((value) => value.trim() !== '');

const decryptMessageRequestBaseSchema = z.object({
  ciphertext: nonEmptyString,
});

export const decryptRsaRequestSchema = decryptMessageRequestBaseSchema.extend({
  method: z.literal('RSA-32768'),
  privateKey: nonEmptyString,
});

export const decryptKuznyechikRequestSchema = decryptMessageRequestBaseSchema.extend({
  method: z.literal('Кузнечик'),
  key: nonEmptyString,
});

export const decryptMessageRequestSchema = z.discriminatedUnion('method', [
  decryptRsaRequestSchema,
  decryptKuznyechikRequestSchema,
]);

export const decryptMessageResultSchema = z.object({
  plaintext: nonEmptyString,
});
