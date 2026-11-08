import {cookies} from 'next/headers';

import type {ExportPdfWithSignatureRequest} from '@/domain/message';

import {appConfig} from '@/appconfig';
import {exportPdfWithSignatureRequestSchema} from '@/infrastructure/http/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';

export const POST = compose(
  auth(),
  castFromJson(exportPdfWithSignatureRequestSchema),
  async (request: ExportPdfWithSignatureRequest) => {
    try {
      const cookie = (await cookies()).toString();

      if (cookie === '') {
        return new Response(null, {status: 401});
      }

      const result = await appConfig
        .resolve('exportPdfWithSignatureService')
        .exportPdfWithSignature(cookie, request);

      if (!result.isSuccess) {
        console.error(result.error);
        return new Response(null, {status: 503});
      }

      return new Response(result.success.content, {
        status: 200,
        headers: {
          'content-type': 'application/pdf',
          'content-disposition': `attachment; filename="${result.success.filename}"`,
        },
      });
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  },
);
