import {CdkVirtualScrollViewport, ScrollingModule} from '@angular/cdk/scrolling';
import {DOCUMENT} from '@angular/common';
import {ChangeDetectionStrategy, Component, DestroyRef, EventEmitter, OnInit, Output, ViewChild, computed, inject, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {HttpClient} from '@angular/common/http';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatSelectModule} from '@angular/material/select';
import {MatTooltipModule} from '@angular/material/tooltip';
import {forkJoin} from 'rxjs';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';

interface EmojiData {
  group?: number;
  hexcode: string;
  label: string;
  order?: number;
  skins?: EmojiData[];
  tags?: string[];
  unicode: string;
}

interface EmojiMessage {
  key: string;
  message: string;
  order?: number;
}

interface EmojiMessages {
  groups: (EmojiMessage & {order: number})[];
  skinTones: EmojiMessage[];
}

interface EmojiGroup extends EmojiMessage {
  icon: string;
  order: number;
}

interface SkinToneOption {
  hexcode: string;
  label: string;
  unicode: string;
}

const EMOJI_COLUMNS = 8;

const SKIN_TONES = [
  {key: 'light', hexcode: '1F3FB', unicode: '🏻'},
  {key: 'medium-light', hexcode: '1F3FC', unicode: '🏼'},
  {key: 'medium', hexcode: '1F3FD', unicode: '🏽'},
  {key: 'medium-dark', hexcode: '1F3FE', unicode: '🏾'},
  {key: 'dark', hexcode: '1F3FF', unicode: '🏿'},
] as const;

@Component({
  selector: 'app-emoji-picker',
  templateUrl: './emoji-picker.component.html',
  styleUrls: ['./emoji-picker.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    MatProgressSpinnerModule,
    MatSelectModule,
    MatTooltipModule,
    ReactiveFormsModule,
    ScrollingModule,
  ],
  standalone: true,
})
export class EmojiPickerComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly document = inject(DOCUMENT);
  private readonly destroyRef = inject(DestroyRef);

  @Output() readonly emojiSelected = new EventEmitter<string>();
  @ViewChild(CdkVirtualScrollViewport) private viewport?: CdkVirtualScrollViewport;

  protected readonly searchControl = new FormControl('', {nonNullable: true});
  protected readonly skinToneControl = new FormControl('', {nonNullable: true});
  protected readonly loading = signal(true);
  protected readonly loadFailed = signal(false);
  protected readonly emojis = signal<EmojiData[]>([]);
  protected readonly messages = signal<EmojiMessages>({groups: [], skinTones: []});
  protected readonly selectedGroup = signal<number | null>(null);
  protected readonly query = signal('');
  protected readonly selectedSkinTone = signal('');

  private readonly normalizedLocale = this.document.documentElement.lang.toLowerCase();
  readonly dataLocale = this.normalizedLocale.startsWith('nb') || this.normalizedLocale.startsWith('no') ? 'nb' : 'en';

  protected readonly groups = computed<EmojiGroup[]>(() => {
    const emojis = this.emojis();
    return this.messages().groups
      .filter(group => emojis.some(emoji => emoji.group === group.order))
      .map(group => ({
        ...group,
        order: group.order,
        icon: emojis.find(emoji => emoji.group === group.order)?.unicode ?? '',
      }))
      .sort((left, right) => left.order - right.order);
  });

  protected readonly skinTones = computed<SkinToneOption[]>(() => {
    const messages = this.messages().skinTones;
    return [
      {hexcode: '', label: $localize`:@@emojiPickerDefaultSkinTone:Default skin tone`, unicode: '👋'},
      ...SKIN_TONES.map(tone => ({
        hexcode: tone.hexcode,
        label: messages.find(message => message.key === tone.key)?.message ?? tone.key,
        unicode: `👋${tone.unicode}`,
      })),
    ];
  });

  protected readonly filteredEmojis = computed(() => {
    const query = this.normalize(this.query());
    const selectedGroup = this.selectedGroup();
    return this.emojis().filter(emoji => {
      if (emoji.group === undefined) {
        return false;
      }
      if (!query) {
        return selectedGroup === null || emoji.group === selectedGroup;
      }
      return this.normalize([emoji.label, ...(emoji.tags ?? [])].join(' ')).includes(query);
    });
  });

  protected readonly emojiRows = computed(() => {
    const rows: EmojiData[][] = [];
    const emojis = this.filteredEmojis();
    for (let index = 0; index < emojis.length; index += EMOJI_COLUMNS) {
      rows.push(emojis.slice(index, index + EMOJI_COLUMNS));
    }
    return rows;
  });

  constructor() {
    this.searchControl.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(value => {
      this.query.set(value);
      this.scrollToTop();
    });
    this.skinToneControl.valueChanges.pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(value => this.selectedSkinTone.set(value));
  }

  ngOnInit(): void {
    this.loadData();
  }

  protected loadData(): void {
    this.loading.set(true);
    this.loadFailed.set(false);
    const basePath = `public/emoji/${this.dataLocale}`;
    forkJoin({
      emojis: this.http.get<EmojiData[]>(`${basePath}/compact.json`),
      messages: this.http.get<EmojiMessages>(`${basePath}/messages.json`),
    }).subscribe({
      next: ({emojis, messages}) => {
        this.emojis.set([...emojis].sort((left, right) => (left.order ?? Number.MAX_SAFE_INTEGER) - (right.order ?? Number.MAX_SAFE_INTEGER)));
        this.messages.set(messages);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.loadFailed.set(true);
      },
    });
  }

  protected selectGroup(group: number | null): void {
    this.selectedGroup.set(group);
    this.searchControl.setValue('');
    this.scrollToTop();
  }

  protected selectEmoji(emoji: EmojiData): void {
    this.emojiSelected.emit(this.displayEmoji(emoji).unicode);
  }

  protected displayEmoji(emoji: EmojiData): EmojiData {
    const skinTone = this.selectedSkinTone();
    if (!skinTone || !emoji.skins?.length) {
      return emoji;
    }
    return emoji.skins.find(skin => skin.hexcode.split('-').includes(skinTone)) ?? emoji;
  }

  protected trackRow(_index: number, row: EmojiData[]): string {
    return row.map(emoji => emoji.hexcode).join(':');
  }

  private normalize(value: string): string {
    return value
      .normalize('NFKD')
      .replace(/\p{Mark}/gu, '')
      .toLocaleLowerCase(this.dataLocale)
      .trim();
  }

  private scrollToTop(): void {
    this.viewport?.scrollToIndex(0);
  }
}
