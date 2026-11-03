import type {ReactNode} from 'react';

import {getUser} from '@/infrastructure/auth/session';

import {Header} from '@/presentation/components/header/Header';

type MainLayoutProps = {
  children: ReactNode;
};

export default async function MainLayout({children}: MainLayoutProps) {
  const user = await getUser();

  return (
    <>
      <Header isAuthorized={user !== null} />
      {children}
    </>
  );
}
