'use client';

import {useMessageField} from '@/presentation/components/message-widget/message-field/useMessageField';

import styles from './MessageField.module.css';

type MessageFieldProps = {
  label: string;
  value: string;
};

export function MessageField({label, value}: MessageFieldProps) {
  const {copied, copy} = useMessageField(value);

  return (
    <div className={styles['message-field']}>
      <div className={styles['message-field__header']}>
        <p className={styles['message-field__label']}>{label}</p>
        <button className={styles['message-field__copy']} type="button" onClick={copy}>
          {copied ? 'Скопировано' : 'Копировать'}
        </button>
      </div>
      <pre className={styles['message-field__value']}>{value}</pre>
    </div>
  );
}
