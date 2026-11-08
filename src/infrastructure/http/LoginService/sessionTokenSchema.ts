import {z} from 'zod';

export const sessionTokenSchema = z.object({
  token: z.string().refine((value) => value.trim() !== ''),
});
