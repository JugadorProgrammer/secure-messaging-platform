import type {ExportPdfWithSignatureRequest} from '@/application/IMessageService/MessageService/ExportPdfWithSignatureRequest';

import {appConfig} from '@/appconfig';
import {exportPdfWithSignatureRequestSchema} from '@/infrastructure/http/MessageService/messageSchema';

import {auth, compose} from '@/presentation/access/auth';
import {castFromJson} from '@/presentation/access/castFromJson';
import {handleErrors} from '@/presentation/access/handleErrors';
import {validateHasCookies} from '@/presentation/access/validateHasCookies';

export const POST = compose(
  auth(),
  validateHasCookies(),
  castFromJson(exportPdfWithSignatureRequestSchema),
  handleErrors(async (request: ExportPdfWithSignatureRequest & {cookie: string}) => {
    const result = await appConfig
      .resolve('messageService')
      .exportPdfWithSignature(request.cookie, request);

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
  }),
);
