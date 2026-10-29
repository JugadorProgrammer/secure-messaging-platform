import {redirect} from 'next/navigation';

const MAIL_URL = 'about:blank';

export function GET() {
  redirect(MAIL_URL);
}
