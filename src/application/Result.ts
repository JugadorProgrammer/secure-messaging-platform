export type Result<T> =
  | {
      isSuccess: true;
      success: T;
    }
  | {
      isSuccess: false;
      error: Error;
    };

export const Result = {
  success<T>(value: T): Result<T> {
    return {
      isSuccess: true,
      success: value,
    };
  },

  error(error: Error): Result<never> {
    return {
      isSuccess: false,
      error,
    };
  },
};
