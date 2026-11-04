import {compose, noAuth} from '@/presentation/access/auth';
import {PasswordRecoveryScreen} from '@/presentation/components/password-recovery-screen/PasswordRecoveryScreen';

export default compose(noAuth, function PasswordRecoveryPage() {
  return <PasswordRecoveryScreen />;
});
