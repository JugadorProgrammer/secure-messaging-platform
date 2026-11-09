import {ADMIN_ROLE} from '@/domain/role';

import {auth, compose} from '@/presentation/access/auth';
import {AdminScreen} from '@/presentation/components/admin-screen/AdminScreen';

export default compose(auth([ADMIN_ROLE]), function AdminPage() {
  return <AdminScreen />;
});
