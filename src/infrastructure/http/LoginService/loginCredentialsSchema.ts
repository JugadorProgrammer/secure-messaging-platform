import {z} from 'zod';

export const loginCredentialsSchema = z.object({
  login: z.string().refine((value) => value.trim() !== ''),
  password: z.string().refine((value) => value.trim() !== ''),
});
