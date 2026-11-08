import type {Message} from '@/domain/message';

import styles from './MessageWidget.module.css';

type MessageWidgetProps = {
  message: Message;
};

export function MessageWidget({}: MessageWidgetProps) {
  return <article className={styles['message-widget']} />;
}
