'use client';

import {useState} from 'react';
import type {ReactElement} from 'react';

import {RecoveryCodeList} from '@/presentation/components/recovery-code-list/RecoveryCodeList';
import {RegistrationCredentialsForm} from '@/presentation/components/registration-credentials-form/RegistrationCredentialsForm';

type UseRegistrationFormParams = {
  initialCodes?: readonly string[];
};

export function useRegistrationForm({
  initialCodes,
}: UseRegistrationFormParams = {}): {
  step: ReactElement;
} {
  const [codes, setCodes] = useState<readonly string[] | null>(initialCodes ?? null);

  const step =
    codes === null ? (
      <RegistrationCredentialsForm onSuccess={setCodes} />
    ) : (
      <RecoveryCodeList codes={codes} />
    );

  return {step};
}
