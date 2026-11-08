export type Result<T, E = Error> =
  | {
      isSuccess: true;
      success: T;
    }
  | {
      isSuccess: false;
      error: E;
    };

export const Result = {
  success<T>(value: T): Result<T, never> {
    return {
      isSuccess: true,
      success: value,
    };
  },

  error<E>(error: E): Result<never, E> {
    return {
      isSuccess: false,
      error,
    };
  },
};
