import {z} from 'zod';

export const restorePasswordRequestSchema = z.object({
  email: z.string().refine((value) => value.trim() !== ''),
});
