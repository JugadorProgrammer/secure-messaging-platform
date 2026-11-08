import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {IRegistrationService} from '@/application/IRegistrationService/IRegistrationService';
import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import type {Result} from '@/application/Result';

type RegistrationServiceDependencies = {
  authGateway: IAuthGateway;
};

export class RegistrationService implements IRegistrationService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: RegistrationServiceDependencies) {
    this.authGateway = authGateway;
  }

  signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>> {
    return this.authGateway.signup(registrationRequest);
  }
}
