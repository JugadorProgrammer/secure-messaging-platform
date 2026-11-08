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

const userSchema = z.object({
  id: z.int().positive(),
  name: nonEmptyString,
});

const messageDtoBaseSchema = z.object({
  id: z.int().positive(),
  user: userSchema,
  plaintext: nonEmptyString,
  ciphertext: nonEmptyString,
  timestamp: z.iso.datetime(),
});

export const rsaMessageDtoSchema = messageDtoBaseSchema.extend({
  method: z.literal('RSA-32768'),
  keys: rsaKeysSchema,
});

export const kuznyechikMessageDtoSchema = messageDtoBaseSchema.extend({
  method: z.literal('Кузнечик'),
  keys: kuznyechikKeysSchema,
});

export const messageDtoSchema = z.discriminatedUnion('method', [
  rsaMessageDtoSchema,
  kuznyechikMessageDtoSchema,
]);

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

export const exportPdfWithSignatureRequestSchema = z.object({
  id: z.int().positive(),
});
