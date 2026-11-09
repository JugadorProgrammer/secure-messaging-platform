import {z} from 'zod';

export const encryptionMethodSchema = z.object({
  name: z.string().refine((value) => value.trim() !== ''),
  enabled: z.boolean(),
});
