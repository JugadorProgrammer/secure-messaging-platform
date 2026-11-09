import {z} from 'zod';

import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

const nonEmptyString = z.string().refine((value) => value.trim() !== '');

export const encryptMessageRequestSchema = z.object({
  plaintext: nonEmptyString,
  method: z.enum(MESSAGE_ENCRYPTION_METHODS),
});

const rsaKeysSchema = z.object({
  publicKey: nonEmptyString,
  privateKey: nonEmptyString,
});

const kuznyechikKeysSchema = z.object({
  key: nonEmptyString,
  iv: nonEmptyString,
});

const messageDtoBaseSchema = z.object({
  id: z.int().positive(),
  user: z.object({
    id: z.int().positive(),
    name: nonEmptyString,
  }),
  plaintext: nonEmptyString,
  ciphertext: nonEmptyString,
  timestamp: z.iso.datetime(),
});

export const messageDtoSchema = z.discriminatedUnion('method', [
  messageDtoBaseSchema.extend({
    method: z.literal('RSA-32768'),
    keys: rsaKeysSchema,
  }),
  messageDtoBaseSchema.extend({
    method: z.literal('Кузнечик'),
    keys: kuznyechikKeysSchema,
  }),
]);

const decryptMessageRequestBaseSchema = z.object({
  ciphertext: nonEmptyString,
});

export const decryptMessageRequestSchema = z.discriminatedUnion('method', [
  decryptMessageRequestBaseSchema.extend({
    method: z.literal('RSA-32768'),
    privateKey: nonEmptyString,
  }),
  decryptMessageRequestBaseSchema.extend({
    method: z.literal('Кузнечик'),
    key: nonEmptyString,
  }),
]);

export const decryptMessageResultSchema = z.object({
  plaintext: nonEmptyString,
});

export const exportPdfWithSignatureRequestSchema = z.object({
  id: z.int().positive(),
});
