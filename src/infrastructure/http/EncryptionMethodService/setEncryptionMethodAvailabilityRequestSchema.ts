import {z} from 'zod';

export const setEncryptionMethodAvailabilityRequestSchema = z.object({
  name: z.string().refine((value) => value.trim() !== ''),
  enabled: z.boolean(),
});
