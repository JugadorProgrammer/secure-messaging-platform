export class ScreenNotice {
  readonly message: string;
  readonly notificationDate = Date.now();
  private holder: NoticeHolder | null = null;

  constructor(message: string) {
    this.message = message;
  }

  attach(holder: NoticeHolder): void {
    if (holder.notice === this) {
      return;
    }

    this.detach();
    holder.notice?.detach();
    this.holder = holder;
    holder.notice = this;
  }

  detach(): void {
    const holder = this.holder;

    if (holder === null) {
      return;
    }

    this.holder = null;

    if (holder.notice === this) {
      holder.notice = null;
    }
  }
}

export class NoticeHolder {
  notice: ScreenNotice | null = null;

  show(message: string): void {
    new ScreenNotice(message).attach(this);
  }

  clear(): void {
    this.notice?.detach();
  }
}
