'use client';

import {useEffect, useState} from 'react';

import type {MessageEncryptionMethod} from '@/domain/message';
import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

import type {DecryptMessageRequest} from '@/application/DecryptMessageRequest';
import {Result} from '@/application/Result';

import {decryptMessage} from '@/presentation/api/decrypt';
import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

type MessageDecryptionWidgetState = {
  ciphertext: string;
  setCiphertext: (value: string) => void;
  privateKey: string;
  setPrivateKey: (value: string) => void;
  key: string;
  setKey: (value: string) => void;
  method: MessageEncryptionMethod | null;
  setMethod: (value: string) => void;
  methods: readonly MessageEncryptionMethod[] | null;
  isMethodsFailed: boolean;
  isSubmitting: boolean;
  plaintext: string | null;
  notice: ReturnType<typeof useScreenNotice>['notice'];
  submit: () => Promise<void>;
};

export function useMessageDecryptionWidget(): MessageDecryptionWidgetState {
  const [ciphertext, setCiphertext] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [key, setKey] = useState('');
  const [method, setMethodState] = useState<MessageEncryptionMethod | null>(null);
  const [methods, setMethods] = useState<readonly MessageEncryptionMethod[] | null>(null);
  const [isMethodsFailed, setIsMethodsFailed] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [plaintext, setPlaintext] = useState<string | null>(null);
  const {notice, show, dismiss} = useScreenNotice();

  useEffect(() => {
    let isCancelled = false;

    void getEnabledEncryptionMethods().then((result) => {
      if (isCancelled) {
        return;
      }

      if (!result.isSuccess) {
        setMethods(null);
        setMethodState(null);
        setIsMethodsFailed(true);
        return;
      }

      const enabledMethods = result.success
        .filter((item) => item.enabled)
        .map((item) => item.name)
        .filter(isMessageEncryptionMethod);

      setMethods(enabledMethods);
      setMethodState(enabledMethods[0] ?? null);
      setIsMethodsFailed(false);
    });

    return () => {
      isCancelled = true;
    };
  }, []);

  function setMethod(value: string): void {
    if (!isMessageEncryptionMethod(value)) {
      return;
    }

    setMethodState(value);
    setPrivateKey('');
    setKey('');
    setPlaintext(null);
  }

  function validate(): Result<DecryptMessageRequest> {
    if (method === null) {
      return Result.error(new Error('Нет доступных методов шифрования.'));
    }

    if (ciphertext.trim() === '') {
      return Result.error(new Error('Введите шифртекст.'));
    }

    if (method === 'RSA-32768') {
      if (privateKey.trim() === '') {
        return Result.error(new Error('Введите private key в формате PEM.'));
      }

      return Result.success({method, ciphertext, privateKey});
    }

    if (key.trim() === '') {
      return Result.error(new Error('Введите ключ.'));
    }

    return Result.success({method, ciphertext, key});
  }

  async function submit(): Promise<void> {
    const validation = validate();

    if (!validation.isSuccess) {
      show(validation.error.message);
      return;
    }

    setIsSubmitting(true);
    dismiss();
    setPlaintext(null);

    const result = await decryptMessage(validation.success);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    setPlaintext(result.success.plaintext);
    show('Сообщение расшифровано.');
  }

  return {
    ciphertext,
    setCiphertext,
    privateKey,
    setPrivateKey,
    key,
    setKey,
    method,
    setMethod,
    methods,
    isMethodsFailed,
    isSubmitting,
    plaintext,
    notice,
    submit,
  };
}

function isMessageEncryptionMethod(value: string): value is MessageEncryptionMethod {
  return (MESSAGE_ENCRYPTION_METHODS as readonly string[]).includes(value);
}
