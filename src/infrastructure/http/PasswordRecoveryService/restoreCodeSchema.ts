import {z} from 'zod';

export const restoreCodeSchema = z.object({
  code: z.string().regex(/^\d{6}$/),
});
