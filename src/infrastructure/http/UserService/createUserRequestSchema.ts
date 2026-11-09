import {z} from 'zod';

import {
  emailPattern,
  loginPattern,
  passwordPattern,
} from '@/infrastructure/http/RegistrationService/registrationRequestSchema';

export const createUserRequestSchema = z.object({
  login: z.string().regex(loginPattern),
  email: z.string().regex(emailPattern),
  password: z.string().regex(passwordPattern),
  role: z.string().refine((value) => value.trim() !== ''),
});
