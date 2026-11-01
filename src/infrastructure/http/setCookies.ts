export function readSetCookies(response: Response): readonly string[] {
  return response.headers.getSetCookie();
}

export function writeSetCookies(response: Response, setCookies: readonly string[]): void {
  for (const setCookie of setCookies) {
    response.headers.append('set-cookie', setCookie);
  }
}
