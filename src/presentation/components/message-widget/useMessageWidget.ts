'use client';

import {useState} from 'react';

import type {DecryptMessageRequest, Message} from '@/domain/message';

import {Result} from '@/application/Result';

import {decryptMessage} from '@/presentation/api/decrypt';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseMessageWidgetState = {
  canCheckFromCiphertext: boolean;
  isChecking: boolean;
  notice: ScreenNotice | null;
  checkFromCiphertext: () => Promise<void>;
};

function toDecryptRequest(message: Message): Result<DecryptMessageRequest> {
  if (message.keys === undefined) {
    return Result.error(new Error('Ключи для сообщения недоступны.'));
  }

  if (message.method === 'RSA-32768') {
    return Result.success({
      method: 'RSA-32768',
      ciphertext: message.ciphertext,
      privateKey: message.keys.privateKey,
    });
  }

  return Result.success({
    method: 'Кузнечик',
    ciphertext: message.ciphertext,
    key: message.keys.key,
  });
}

export function useMessageWidget(message: Message): UseMessageWidgetState {
  const [isChecking, setIsChecking] = useState(false);
  const {notice, show, dismiss} = useScreenNotice();
  const canCheckFromCiphertext = message.keys !== undefined;

  async function checkFromCiphertext(): Promise<void> {
    const request = toDecryptRequest(message);

    if (!request.isSuccess) {
      show(request.error.message);
      return;
    }

    setIsChecking(true);
    dismiss();

    const result = await decryptMessage(request.success);

    setIsChecking(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    if (result.success.plaintext === message.plaintext) {
      show('Проверка из шифртекста пройдена.');
      return;
    }

    show('Проверка из шифртекста не пройдена.');
  }

  return {
    canCheckFromCiphertext,
    isChecking,
    notice,
    checkFromCiphertext,
  };
}
