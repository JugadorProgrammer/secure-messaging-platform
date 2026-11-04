import {appConfig} from '@/appconfig';

export async function GET(): Promise<Response> {
  try {
    const result = await appConfig.resolve('roleService').getRoles();

    if (!result.isSuccess) {
      console.error(result.error);
      return new Response(null, {status: 503});
    }

    return Response.json(result.success);
  } catch (error) {
    console.error(error);
    return new Response(null, {status: 503});
  }
}
