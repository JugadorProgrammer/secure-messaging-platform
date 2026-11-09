import {z} from 'zod';

export const disposableCodeRequestSchema = z.object({
  code: z.string().refine((value) => value.trim() !== ''),
});
