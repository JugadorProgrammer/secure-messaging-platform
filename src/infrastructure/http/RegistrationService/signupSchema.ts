import {z} from 'zod';

export const loginPattern = /^[A-Za-z][A-Za-z0-9]{2,31}$/;
export const emailPattern = /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/;
export const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,64}$/;

export const signupSchema = z.object({
  login: z.string().regex(loginPattern),
  email: z.string().regex(emailPattern),
  password: z.string().regex(passwordPattern),
});
