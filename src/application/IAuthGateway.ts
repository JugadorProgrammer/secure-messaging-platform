import type {Result} from '@/application/Result';

export type LoginCredentials = {
  login: string;
  password: string;
};

export type EmailCode = {
  code: string;
};

export interface IAuthGateway {
  login(credentials: LoginCredentials): Promise<Result<readonly string[]>>;
  confirmEmailCode(cookie: string, emailCode: EmailCode): Promise<Result<readonly string[]>>;
}
