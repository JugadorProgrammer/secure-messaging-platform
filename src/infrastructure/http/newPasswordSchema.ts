import {z} from 'zod';

export const newPasswordSchema = z.object({
  password: z.string().refine((value) => value.trim() !== ''),
});
