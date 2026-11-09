import {z} from 'zod';

export const emailCodeRequestSchema = z.object({
  code: z.string().regex(/^\d{6}$/),
});
