import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {ILoginService} from '@/application/ILoginService/ILoginService';
import type {DisposableCode} from '@/application/ILoginService/LoginService/DisposableCode';
import type {EmailCode} from '@/application/ILoginService/LoginService/EmailCode';
import type {LoginCredentials} from '@/application/ILoginService/LoginService/LoginCredentials';
import type {Result} from '@/application/Result';

type LoginServiceDependencies = {
  authGateway: IAuthGateway;
};

export class LoginService implements ILoginService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: LoginServiceDependencies) {
    this.authGateway = authGateway;
  }

  login(credentials: LoginCredentials): Promise<Result<readonly string[]>> {
    return this.authGateway.login(credentials);
  }

  confirmEmailCode(cookie: string, emailCode: EmailCode): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmEmailCode(cookie, emailCode);
  }

  confirmDisposableCode(
    cookie: string,
    disposableCode: DisposableCode,
  ): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmDisposableCode(cookie, disposableCode);
  }
}
