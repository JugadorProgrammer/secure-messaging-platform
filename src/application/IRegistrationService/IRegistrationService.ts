import type {RegistrationRequest} from '@/application/IRegistrationService/RegistrationService/RegistrationRequest';
import type {Result} from '@/application/Result';

export interface IRegistrationService {
  signup(registrationRequest: RegistrationRequest): Promise<Result<string[]>>;
}
