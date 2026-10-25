import {appConfig} from '@/appconfig';

export async function GET(): Promise<Response> {
  try {
    const result = await appConfig.resolve('roleService').getRoles();

    if (!result.isSuccess) {
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  } catch {
    return new Response(null, {status: 503});
  }
}
