export function handleErrors(
  handler: (...args: never[]) => Response | Promise<Response>,
) {
  return async (...args: unknown[]): Promise<Response> => {
    try {
      return await (handler as (...args: unknown[]) => Promise<Response>)(...args);
    } catch (error) {
      console.error(error);
      return new Response(null, {status: 503});
    }
  };
}
