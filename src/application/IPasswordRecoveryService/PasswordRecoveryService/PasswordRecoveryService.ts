import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {IPasswordRecoveryService} from '@/application/IPasswordRecoveryService/IPasswordRecoveryService';
import type {NewPassword} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPassword';
import type {RestoreCode} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCode';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {Result} from '@/application/Result';

export const RESTORE_PASSWORD_COOKIE = 'restorePassword';

type PasswordRecoveryServiceDependencies = {
  authGateway: IAuthGateway;
};

export class PasswordRecoveryService implements IPasswordRecoveryService {
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
