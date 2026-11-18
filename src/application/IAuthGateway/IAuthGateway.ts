import type {DisposableCodeRequest} from '@/application/ILoginService/LoginService/DisposableCodeRequest';
import type {EmailCodeRequest} from '@/application/ILoginService/LoginService/EmailCodeRequest';
import type {LoginCredentialsRequest} from '@/application/ILoginService/LoginService/LoginCredentialsRequest';
import type {NewPasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/NewPasswordRequest';
import type {RestoreCodeRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestoreCodeRequest';
import type {RestorePasswordRequest} from '@/application/IPasswordRecoveryService/PasswordRecoveryService/RestorePasswordRequest';
import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import type {CreateUserRequest} from '@/application/IUserService/UserService/CreateUserRequest';
import type {UserDto} from '@/application/IUserService/UserService/UserDto';
import type {Result} from '@/application/Result';

export interface IAuthGateway {
  login(loginCredentialsRequest: LoginCredentialsRequest): Promise<Result<readonly string[]>>;
  confirmEmailCode(
    cookie: string,
    emailCodeRequest: EmailCodeRequest,
  ): Promise<Result<readonly string[]>>;
  confirmDisposableCode(
    cookie: string,
    disposableCodeRequest: DisposableCodeRequest,
  ): Promise<Result<readonly string[]>>;
  restorePassword(restorePasswordRequest: RestorePasswordRequest): Promise<Result<void>>;
  confirmRestoreCode(restoreCodeRequest: RestoreCodeRequest): Promise<Result<readonly string[]>>;
  confirmNewPassword(
    cookie: string,
    newPasswordRequest: NewPasswordRequest,
  ): Promise<Result<readonly string[]>>;
  signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>>;
  getUsers(cookie: string): Promise<Result<UserDto[]>>;
  createUser(cookie: string, request: CreateUserRequest): Promise<Result<string[]>>;
}
