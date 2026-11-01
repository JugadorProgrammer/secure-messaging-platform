import {z} from 'zod';

export const roleSchema = z.object({
  id: z.int(),
  name: z.string().refine((value) => value.trim() !== ''),
  description: z.string().refine((value) => value.trim() !== ''),
});
