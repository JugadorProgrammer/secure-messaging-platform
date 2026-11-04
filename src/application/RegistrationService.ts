import type {IAuthGateway, Registration} from '@/application/IAuthGateway';
import type {Result} from '@/application/Result';

type RegistrationServiceDependencies = {
  authGateway: IAuthGateway;
};

export class RegistrationService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: RegistrationServiceDependencies) {
    this.authGateway = authGateway;
  }

  signup(registration: Registration): Promise<Result<string[]>> {
    return this.authGateway.signup(registration);
  }
}
