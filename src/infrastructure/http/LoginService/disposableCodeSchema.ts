import {z} from 'zod';

export const disposableCodeSchema = z.object({
  code: z.string().refine((value) => value.trim() !== ''),
});
