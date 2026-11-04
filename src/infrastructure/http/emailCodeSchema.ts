import {z} from 'zod';

export const emailCodeSchema = z.object({
  code: z.string().regex(/^\d{6}$/),
});
