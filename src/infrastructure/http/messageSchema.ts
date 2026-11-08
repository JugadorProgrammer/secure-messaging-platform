import {z} from 'zod';

import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

export const messageEncryptionMethodSchema = z.enum(MESSAGE_ENCRYPTION_METHODS);

const nonEmptyString = z.string().refine((value) => value.trim() !== '');

export const encryptMessageRequestSchema = z.object({
  plaintext: nonEmptyString,
  method: messageEncryptionMethodSchema,
});

const rsaKeysSchema = z.object({
  publicKey: nonEmptyString,
  privateKey: nonEmptyString,
});

const kuznyechikKeysSchema = z.object({
  key: nonEmptyString,
  iv: nonEmptyString,
});

const messageBaseSchema = z.object({
  plaintext: nonEmptyString,
  ciphertext: nonEmptyString,
  timestamp: z.iso.datetime(),
});

export const rsaMessageSchema = messageBaseSchema.extend({
  method: z.literal('RSA-32768'),
  keys: rsaKeysSchema,
});

export const kuznyechikMessageSchema = messageBaseSchema.extend({
  method: z.literal('Кузнечик'),
  keys: kuznyechikKeysSchema,
});

export const messageSchema = z.discriminatedUnion('method', [
  rsaMessageSchema,
  kuznyechikMessageSchema,
]);
