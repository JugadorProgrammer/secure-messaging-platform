import type {NewPassword} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPassword';
import type {RestoreCode} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCode';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {Result} from '@/application/Result';

export interface IPasswordRecoveryService {
  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>>;
  confirmRestoreCode(restoreCode: RestoreCode): Promise<Result<readonly string[]>>;
  confirmNewPassword(cookie: string, newPassword: NewPassword): Promise<Result<readonly string[]>>;
}
