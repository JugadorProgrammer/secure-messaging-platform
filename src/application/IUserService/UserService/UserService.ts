import type {IAuthGateway} from '@/application/IAuthGateway/IAuthGateway';
import type {IUserService} from '@/application/IUserService/IUserService';
import type {UserDto} from '@/application/IUserService/UserService/UserDto';
import type {Result} from '@/application/Result';

type UserServiceDependencies = {
  authGateway: IAuthGateway;
};

export class UserService implements IUserService {
  private readonly authGateway: IAuthGateway;

  constructor({authGateway}: UserServiceDependencies) {
    this.authGateway = authGateway;
  }

  getUsers(cookie: string): Promise<Result<UserDto[]>> {
    return this.authGateway.getUsers(cookie);
  }
}
