import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {ILoginService} from '@/application/ILoginService/ILoginService';
import type {DisposableCodeRequest} from '@/application/ILoginService/LoginService/DisposableCodeRequest';
import type {EmailCodeRequest} from '@/application/ILoginService/LoginService/EmailCodeRequest';
import type {LoginCredentialsRequest} from '@/application/ILoginService/LoginService/LoginCredentialsRequest';
import type {Result} from '@/application/Result';

type LoginServiceDependencies = {
  authGateway: IAuthGateway;
};

export class LoginService implements ILoginService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: LoginServiceDependencies) {
    this.authGateway = authGateway;
  }

  login(loginCredentialsRequest: LoginCredentialsRequest): Promise<Result<readonly string[]>> {
    return this.authGateway.login(loginCredentialsRequest);
  }

  confirmEmailCode(
    cookie: string,
    emailCodeRequest: EmailCodeRequest,
  ): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmEmailCode(cookie, emailCodeRequest);
  }

  confirmDisposableCode(
    cookie: string,
    disposableCodeRequest: DisposableCodeRequest,
  ): Promise<Result<readonly string[]>> {
    return this.authGateway.confirmDisposableCode(cookie, disposableCodeRequest);
  }
}
