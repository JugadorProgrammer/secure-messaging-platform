import {compose, noAuth} from '@/presentation/access/auth';
import {RegistrationScreen} from '@/presentation/components/registration-screen/RegistrationScreen';

export default compose(noAuth, function RegistrationPage() {
  return <RegistrationScreen />;
});
