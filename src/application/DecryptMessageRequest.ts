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
