import type {ExportPdfWithSignatureRequest, SignedPdfDocument} from '@/domain/message';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type ExportPdfWithSignatureServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class ExportPdfWithSignatureService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: ExportPdfWithSignatureServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  exportPdfWithSignature(
    cookie: string,
    request: ExportPdfWithSignatureRequest,
  ): Promise<Result<SignedPdfDocument>> {
    return this.coordGateway.exportPdfWithSignature(cookie, request);
  }
}
