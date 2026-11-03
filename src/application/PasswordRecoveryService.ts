import type {IAuthGateway, NewPassword, RestoreCode, RestorePasswordRequest} from '@/application/IAuthGateway';
import type {Result} from '@/application/Result';

export const RESTORE_PASSWORD_COOKIE = 'restorePassword';

type PasswordRecoveryServiceDependencies = {
  authGateway: IAuthGateway;
};

export class PasswordRecoveryService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: PasswordRecoveryServiceDependencies) {
    this.authGateway = authGateway;
  }

  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>> {
    return this.authGateway.restorePassword(restorePasswordRequest);
  }

  confirmRestoreCode(restoreCode: RestoreCode): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmRestoreCode(restoreCode);
  }

  confirmNewPassword(cookie: string, newPassword: NewPassword): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmNewPassword(cookie, newPassword);
  }
}
