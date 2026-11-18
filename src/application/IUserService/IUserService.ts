import type {CreateUserRequest} from '@/application/IUserService/UserService/CreateUserRequest';
import type {UserDto} from '@/application/IUserService/UserService/UserDto';
import type {Result} from '@/application/Result';

export interface IUserService {
  getUsers(cookie: string): Promise<Result<UserDto[]>>;
  createUser(cookie: string, request: CreateUserRequest): Promise<Result<string[]>>;
}
