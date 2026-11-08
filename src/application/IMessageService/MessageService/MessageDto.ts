import type {KuznyechikKeys, RsaKeys} from '@/domain/message';
import type {User} from '@/domain/user';

type MessageDtoBase = {
  id: number;
  user: User;
  plaintext: string;
  ciphertext: string;
  timestamp: string;
};

export type RsaMessageDto = MessageDtoBase & {
  method: 'RSA-32768';
  keys: RsaKeys;
};

export type KuznyechikMessageDto = MessageDtoBase & {
  method: 'Кузнечик';
  keys: KuznyechikKeys;
};

export type MessageDto = RsaMessageDto | KuznyechikMessageDto;
