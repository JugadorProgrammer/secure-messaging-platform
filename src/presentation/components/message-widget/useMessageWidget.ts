'use client';

import {useState} from 'react';

import type {DecryptMessageRequest, Message, SignedPdfDocument} from '@/domain/message';

import {Result} from '@/application/Result';

import {decryptMessage} from '@/presentation/api/decrypt';
import {exportPdfWithSignature} from '@/presentation/api/exportPdfWithSignature';
import {useScreenNotice} from '@/presentation/notice/useScreenNotice';
import type {ScreenNotice} from '@/presentation/notice/ScreenNotice';

type UseMessageWidgetState = {
  canCheckFromCiphertext: boolean;
  isChecking: boolean;
  isExportingPdf: boolean;
  notice: ScreenNotice | null;
  checkFromCiphertext: () => Promise<void>;
  exportPdf: () => Promise<void>;
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

function downloadPdf(pdf: SignedPdfDocument): void {
  const blob = new Blob([pdf.content], {type: 'application/pdf'});
  const url = URL.createObjectURL(blob);
  const link = window.document.createElement('a');

  link.href = url;
  link.download = pdf.filename;
  link.click();
  URL.revokeObjectURL(url);
}

export function useMessageWidget(message: Message): UseMessageWidgetState {
  const [isChecking, setIsChecking] = useState(false);
  const [isExportingPdf, setIsExportingPdf] = useState(false);
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

  async function exportPdf(): Promise<void> {
    setIsExportingPdf(true);
    dismiss();

    const result = await exportPdfWithSignature({id: message.id});

    setIsExportingPdf(false);

    if (!result.isSuccess) {
      show(result.error.message);
      return;
    }

    downloadPdf(result.success);
    show('PDF с ЭЦП сформирован.');
  }

  return {
    canCheckFromCiphertext,
    isChecking,
    isExportingPdf,
    notice,
    checkFromCiphertext,
    exportPdf,
  };
}
