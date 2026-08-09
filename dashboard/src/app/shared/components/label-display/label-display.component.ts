import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import emojiRegex from 'emoji-regex';

export const EMOJI_LABEL_KEY = 'emoji';

export function isSingleEmoji(value: string): boolean {
  const matches = value?.match(emojiRegex());
  return matches?.length === 1 &&
    stripPresentationSelectors(matches[0]) === stripPresentationSelectors(value);
}

function stripPresentationSelectors(value: string): string {
  return value.replace(/[\uFE0E\uFE0F]/g, '');
}

export function isEmojiLabel(key: string, value: string): boolean {
  return key === EMOJI_LABEL_KEY && isSingleEmoji(value);
}

@Component({
  selector: 'app-label-display',
  templateUrl: './label-display.component.html',
  styleUrls: ['./label-display.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class LabelDisplayComponent {
  @Input({required: true}) key: string;
  @Input({required: true}) value: string;

  get rawLabel(): string {
    return `${this.key}:${this.value}`;
  }

  get emoji(): boolean {
    return isEmojiLabel(this.key, this.value);
  }
}
