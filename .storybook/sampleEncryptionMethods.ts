import type {EncryptionMethod} from '@/domain/encryptionMethod';

export const sampleEncryptionMethods: EncryptionMethod[] = [
  {
    name: 'RSA-32768',
    enabled: true,
  },
  {
    name: 'Кузнечик',
    enabled: false,
  },
];
