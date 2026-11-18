import {z} from 'zod';

export const userDtoSchema = z.object({
  id: z.int(),
  login: z.string().refine((value) => value.trim() !== ''),
  email: z.string().refine((value) => value.trim() !== ''),
  role: z.string().refine((value) => value.trim() !== ''),
});
