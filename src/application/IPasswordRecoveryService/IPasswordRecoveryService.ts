import type {NewPasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPasswordRequest';
import type {RestoreCodeRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCodeRequest';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {Result} from '@/application/Result';

export interface IPasswordRecoveryService {
  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>>;
  confirmRestoreCode(restoreCodeRequest: RestoreCodeRequest): Promise<Result<readonly string[]>>;
  confirmNewPassword(
    cookie: string,
    newPasswordRequest: NewPasswordRequest,
  ): Promise<Result<readonly string[]>>;
}
