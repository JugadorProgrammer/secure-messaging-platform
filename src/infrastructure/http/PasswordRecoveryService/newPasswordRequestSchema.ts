import {z} from 'zod';

export const newPasswordRequestSchema = z.object({
  password: z.string().refine((value) => value.trim() !== ''),
});
