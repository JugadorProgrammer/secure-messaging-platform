import type {ExportPdfWithSignatureRequest} from '@/application/ExportPdfWithSignatureRequest';
import type {SignedPdfDocument} from '@/application/SignedPdfDocument';

import {Result} from '@/application/Result';

import {HttpRequestBuilder} from '@/infrastructure/http/HttpRequestBuilder';

const EXPORT_PDF_WITH_SIGNATURE_PATH = '/api/exportPdfWithSignature';
const DEFAULT_PDF_FILENAME = 'message.pdf';

export async function exportPdfWithSignature(
  request: ExportPdfWithSignatureRequest,
): Promise<Result<SignedPdfDocument>> {
  try {
    const response = await fetch(
      EXPORT_PDF_WITH_SIGNATURE_PATH,
      new HttpRequestBuilder().post().json(request).build(),
    );

    if (!response.ok) {
      return Result.error(new Error('Не удалось сформировать PDF с ЭЦП.'));
    }

    const content = await response.arrayBuffer();

    if (content.byteLength === 0) {
      return Result.error(new Error('Не удалось сформировать PDF с ЭЦП.'));
    }

    return Result.success({
      content,
      filename: readFilename(response.headers.get('content-disposition')) ?? DEFAULT_PDF_FILENAME,
    });
  } catch {
    return Result.error(new Error('Не удалось сформировать PDF с ЭЦП.'));
  }
}

function readFilename(header: string | null): string | null {
  if (header === null) {
    return null;
  }

  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(header);

  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return utf8Match[1];
    }
  }

  const basicMatch = /filename="?([^";]+)"?/i.exec(header);

  return basicMatch?.[1] ?? null;
}
