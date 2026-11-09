import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {IPasswordRecoveryService} from '@/application/IPasswordRecoveryService/IPasswordRecoveryService';
import type {NewPasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPasswordRequest';
import type {RestoreCodeRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCodeRequest';
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

  confirmRestoreCode(restoreCodeRequest: RestoreCodeRequest): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmRestoreCode(restoreCodeRequest);
  }

  confirmNewPassword(
    cookie: string,
    newPasswordRequest: NewPasswordRequest,
  ): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmNewPassword(cookie, newPasswordRequest);
  }
}
