import type {Role} from '@/domain/role';

import type {ICoordGateway} from '@/application/ICoordGateway';
import type {Result} from '@/application/Result';

type RoleServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class RoleService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: RoleServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getRoles(): Promise<Result<Role[]>> {
    return this.coordGateway.getRoles();
  }
}
