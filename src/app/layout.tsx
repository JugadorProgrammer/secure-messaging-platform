import type {Metadata} from 'next';
import type {ReactNode} from 'react';

import './globals.css';

export const metadata: Metadata = {
  title: 'Secure Messaging Platform',
  description: 'Визуальный интерфейс Secure Messaging Platform',
};

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({children}: RootLayoutProps) {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
