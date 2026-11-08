export const MESSAGE_ENCRYPTION_METHODS = ['RSA-32768', 'Кузнечик'] as const;

export type MessageEncryptionMethod = (typeof MESSAGE_ENCRYPTION_METHODS)[number];

type DecryptMessageRequestBase = {
  ciphertext: string;
};

export type DecryptRsaRequest = DecryptMessageRequestBase & {
  method: 'RSA-32768';
  privateKey: string;
};

export type DecryptKuznyechikRequest = DecryptMessageRequestBase & {
  method: 'Кузнечик';
  key: string;
};

export type DecryptMessageRequest = DecryptRsaRequest | DecryptKuznyechikRequest;

export type DecryptMessageResult = {
  plaintext: string;
};
