import type {ICoordGateway, LoginCredentials} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type LoginServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class LoginService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: LoginServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  login(credentials: LoginCredentials): Promise<Result<{token: string}>> {
    return this.coordGateway.login(credentials);
  }
}
