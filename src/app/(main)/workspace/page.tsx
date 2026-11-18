import {auth, compose} from '@/presentation/access/auth';
import {WorkspaceScreen} from '@/presentation/components/workspace-screen/WorkspaceScreen';

export default compose(auth(), function WorkspacePage() {
  return <WorkspaceScreen />;
});
