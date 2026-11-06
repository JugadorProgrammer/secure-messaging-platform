export class HttpRequestBuilder {
  private method = 'GET';
  private readonly headers: Record<string, string> = {};
  private body: string | undefined;

  post(): this {
    this.method = 'POST';
    return this;
  }

  json(payload: unknown): this {
    this.headers['content-type'] = 'application/json';
    this.body = JSON.stringify(payload);
    return this;
  }

  cookie(cookie: string): this {
    this.headers.cookie = cookie;
    return this;
  }

  build(): RequestInit {
    const init: RequestInit = {
      method: this.method,
      headers: {...this.headers},
    };

    if (this.body !== undefined) {
      init.body = this.body;
    }

    return init;
  }
}
