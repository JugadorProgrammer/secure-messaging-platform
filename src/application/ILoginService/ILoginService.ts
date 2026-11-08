import type {DisposableCode} from '@/application/ILoginService/LoginService/DisposableCode';
import type {EmailCode} from '@/application/ILoginService/LoginService/EmailCode';
import type {LoginCredentials} from '@/application/ILoginService/LoginService/LoginCredentials';
import type {Result} from '@/application/Result';

export interface ILoginService {
  login(credentials: LoginCredentials): Promise<Result<readonly string[]>>;
  confirmEmailCode(cookie: string, emailCode: EmailCode): Promise<Result<readonly string[]>>;
  confirmDisposableCode(
    cookie: string,
    disposableCode: DisposableCode,
  ): Promise<Result<readonly string[]>>;
}
