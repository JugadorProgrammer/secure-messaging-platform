import type {Role} from '@/domain/role';

import type {Result} from '@/application/Result';

export interface IRoleService {
  getRoles(): Promise<Result<Role[]>>;
}
