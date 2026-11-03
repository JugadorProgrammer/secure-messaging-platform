import {z} from 'zod';

const recoveryCodeSchema = z.string().refine((value) => value.trim() !== '');

export const recoveryCodesSchema = z.array(recoveryCodeSchema);
