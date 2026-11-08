export const MESSAGE_ENCRYPTION_METHODS = ['RSA-32768', 'Кузнечик'] as const;

export type MessageEncryptionMethod = (typeof MESSAGE_ENCRYPTION_METHODS)[number];

export type RsaKeys = {
  publicKey: string;
  privateKey: string;
};

export type KuznyechikKeys = {
  key: string;
  iv: string;
};

type MessageBase = {
  id: number;
  plaintext: string;
  ciphertext: string;
  timestamp: string;
};

export type RsaMessage = MessageBase & {
  method: 'RSA-32768';
  keys: RsaKeys;
};

export type KuznyechikMessage = MessageBase & {
  method: 'Кузнечик';
  keys: KuznyechikKeys;
};

export type Message = RsaMessage | KuznyechikMessage;
