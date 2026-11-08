import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {IRegistrationService} from '@/application/IRegistrationService/IRegistrationService';
import type {Registration} from '@/application/IRegistrationService/RegistrationService/Registration';
import type {Result} from '@/application/Result';

type RegistrationServiceDependencies = {
  authGateway: IAuthGateway;
};

export class RegistrationService implements IRegistrationService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: RegistrationServiceDependencies) {
    this.authGateway = authGateway;
  }

  signup(registration: Registration): Promise<Result<string[]>> {
    return this.authGateway.signup(registration);
  }
}
