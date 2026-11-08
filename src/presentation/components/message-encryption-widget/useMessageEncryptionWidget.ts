'use client';

import {useEffect, useState} from 'react';

import type {MessageEncryptionMethod} from '@/domain/message';
import {MESSAGE_ENCRYPTION_METHODS} from '@/domain/message';

import type {MessageDto} from '@/application/IMessageService/MessageService/MessageDto';
import {Result} from '@/application/Result';

import {encryptMessage} from '@/presentation/api/encrypt';
import {getEnabledEncryptionMethods} from '@/presentation/api/encryptionMethods';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';

type UseMessageEncryptionWidgetParams = {
  onEncrypted: (message: MessageDto) => void;
};

type MessageEncryptionWidgetState = {
  plaintext: string;
  setPlaintext: (value: string) => void;
  method: MessageEncryptionMethod | null;
  setMethod: (value: string) => void;
  methods: readonly MessageEncryptionMethod[] | null;
  isMethodsFailed: boolean;
  isSubmitting: boolean;
  notice: ReturnType<typeof useScreenNotice>['notice'];
  submit: () => Promise<void>;
};

export function useMessageEncryptionWidget({
  onEncrypted,
}: UseMessageEncryptionWidgetParams): MessageEncryptionWidgetState {
  const [plaintext, setPlaintext] = useState('');
  const [method, setMethodState] = useState<MessageEncryptionMethod | null>(null);
  const [methods, setMethods] = useState<readonly MessageEncryptionMethod[] | null>(null);
  const [isMethodsFailed, setIsMethodsFailed] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
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
    if (isMessageEncryptionMethod(value)) {
      setMethodState(value);
    }
  }

  function validate(): Result<{plaintext: string; method: MessageEncryptionMethod}> {
    if (plaintext.trim() === '') {
      return Result.error(new Error('Введите исходный текст.'));
    }

    if (method === null) {
      return Result.error(new Error('Нет доступных методов шифрования.'));
    }

    return Result.success({plaintext, method});
  }

  async function submit(): Promise<void> {
    const validation = validate();

    if (!validation.isSuccess) {
      show(validation.error.message);
      return;
    }

    setIsSubmitting(true);
    dismiss();

    const result = await encryptMessage(validation.success);

    setIsSubmitting(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    setPlaintext('');
    onEncrypted(result.success);
    show('Сообщение сохранено и зашифровано.');
  }

  return {
    plaintext,
    setPlaintext,
    method,
    setMethod,
    methods,
    isMethodsFailed,
    isSubmitting,
    notice,
    submit,
  };
}

function isMessageEncryptionMethod(value: string): value is MessageEncryptionMethod {
  return (MESSAGE_ENCRYPTION_METHODS as readonly string[]).includes(value);
}
