import {compose, noAuth} from '@/presentation/access/auth';
import {LoginScreen} from '@/presentation/components/login-screen/LoginScreen';

export default compose(noAuth, function LoginPage() {
  return <LoginScreen />;
});
