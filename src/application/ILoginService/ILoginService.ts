import type {DisposableCodeRequest} from '@/application/ILoginService/LoginService/DisposableCodeRequest';
import type {EmailCodeRequest} from '@/application/ILoginService/LoginService/EmailCodeRequest';
import type {LoginCredentialsRequest} from '@/application/ILoginService/LoginService/LoginCredentialsRequest';
import type {Result} from '@/application/Result';

export interface ILoginService {
  login(loginCredentialsRequest: LoginCredentialsRequest): Promise<Result<readonly string[]>>;
  confirmEmailCode(
    cookie: string,
    emailCodeRequest: EmailCodeRequest,
  ): Promise<Result<readonly string[]>>;
  confirmDisposableCode(
    cookie: string,
    disposableCodeRequest: DisposableCodeRequest,
  ): Promise<Result<readonly string[]>>;
}
