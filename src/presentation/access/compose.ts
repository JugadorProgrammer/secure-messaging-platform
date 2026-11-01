export function compose<Result>(
  ...functions: [
    ...Array<(...args: unknown[]) => unknown>,
    (...args: unknown[]) => Result | Promise<Result>,
  ]
) {
  return (...initial: unknown[]): Promise<Awaited<Result>> =>
    functions.reduce<Promise<unknown>>(
      (result, fn) =>
        result.then((value) => {
          if (value instanceof Response) {
            return value;
          }

          return fn(...(value as unknown[]));
        }),
      Promise.resolve(initial),
    ) as Promise<Awaited<Result>>;
}
