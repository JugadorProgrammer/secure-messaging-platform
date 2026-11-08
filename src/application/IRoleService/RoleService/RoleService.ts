import type {Role} from '@/domain/role';

import type {ICoordGateway} from '@/application/ICoordGateway/ICoordGateway';
import type {IRoleService} from '@/application/IRoleService/IRoleService';
import type {Result} from '@/application/Result';

type RoleServiceDependencies = {
  coordGateway: ICoordGateway;
};

export class RoleService implements IRoleService {
  private readonly coordGateway: ICoordGateway;

  constructor({coordGateway}: RoleServiceDependencies) {
    this.coordGateway = coordGateway;
  }

  getRoles(): Promise<Result<Role[]>> {
    return this.coordGateway.getRoles();
  }
}
