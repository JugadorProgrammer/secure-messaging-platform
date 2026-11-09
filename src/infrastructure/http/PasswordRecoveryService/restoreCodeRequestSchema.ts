import {z} from 'zod';

export const restoreCodeRequestSchema = z.object({
  code: z.string().regex(/^\d{6}$/),
});
