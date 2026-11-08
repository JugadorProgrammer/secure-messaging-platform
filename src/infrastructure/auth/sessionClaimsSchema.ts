import {z} from 'zod';

export const sessionClaimsSchema = z.object({
  id: z.int().positive(),
  name: z.string().refine((value) => value.trim() !== ''),
  roles: z.array(z.string()),
});
