import type {Registration} from '@/application/IRegistrationService/RegistrationService/Registration';
import type {Result} from '@/application/Result';

export interface IRegistrationService {
  signup(registration: Registration): Promise<Result<string[]>>;
}
