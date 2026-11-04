import type {DisposableCode, EmailCode, IAuthGateway, LoginCredentials} from '@/application/IAuthGateway';
import type {Result} from '@/application/Result';

type LoginServiceDependencies = {
  authGateway: IAuthGateway;
};

export class LoginService {
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

  confirmDisposableCode(cookie: string, disposableCode: DisposableCode): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmDisposableCode(cookie, disposableCode);
  }
}
