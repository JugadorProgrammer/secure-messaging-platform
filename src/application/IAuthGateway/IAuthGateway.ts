import type {DisposableCode} from '@/application/ILoginService/LoginService/DisposableCode';
import type {EmailCode} from '@/application/ILoginService/LoginService/EmailCode';
import type {LoginCredentials} from '@/application/ILoginService/LoginService/LoginCredentials';
import type {NewPassword} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPassword';
import type {RestoreCode} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCode';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {Registration} from '@/application/IRegistrationService/RegistrationService/Registration';
import type {Result} from '@/application/Result';

export interface IAuthGateway {
  login(credentials: LoginCredentials): Promise<Result<readonly string[]>>;
  confirmEmailCode(cookie: string, emailCode: EmailCode): Promise<Result<readonly string[]>>;
  confirmDisposableCode(
    cookie: string,
    disposableCode: DisposableCode,
  ): Promise<Result<readonly string[]>>;
  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>>;
  confirmRestoreCode(restoreCode: RestoreCode): Promise<Result<readonly string[]>>;
  confirmNewPassword(cookie: string, newPassword: NewPassword): Promise<Result<readonly string[]>>;
  signup(registration: Registration): Promise<Result<string[]>>;
}
