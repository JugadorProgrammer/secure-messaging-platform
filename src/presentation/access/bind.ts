export function bind<Args extends unknown[]>(
  args: readonly unknown[],
  values: object,
): Args {
  return args.map((arg) => {
    if (typeof Request !== 'undefined' && arg instanceof Request) {
      return Object.assign(arg, values);
    }

    if (typeof arg === 'object' && arg !== null) {
      return {...arg, ...values};
    }

    return arg;
  }) as Args;
}
