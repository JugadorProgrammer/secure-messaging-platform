import type {MessageEncryptionMethod} from '@/domain/message';

export type EncryptMessageRequest = {
  plaintext: string;
  method: MessageEncryptionMethod;
};
