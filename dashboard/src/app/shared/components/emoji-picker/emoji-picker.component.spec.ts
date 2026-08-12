import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatSelectHarness} from '@angular/material/select/testing';
import {MatTooltipHarness} from '@angular/material/tooltip/testing';

import {EmojiPickerComponent} from './emoji-picker.component';

interface EmojiPickerTestAccess {
  groups: () => {message: string}[];
  searchControl: {setValue(value: string): void};
  filteredEmojis: () => {unicode: string}[];
  skinToneControl: {setValue(value: string): void};
  displayEmoji(emoji: unknown): {unicode: string};
  selectEmoji(emoji: unknown): void;
}

const emojiData = [
  {
    group: 0,
    hexcode: '1F600',
    label: 'grinning face',
    order: 1,
    tags: ['face', 'smile'],
    unicode: '😀',
  },
  {
    group: 1,
    hexcode: '1F44B',
    label: 'waving hand',
    order: 2,
    tags: ['hand', 'wave'],
    unicode: '👋',
    skins: [{
      group: 1,
      hexcode: '1F44B-1F3FD',
      label: 'waving hand: medium skin tone',
      order: 3,
      unicode: '👋🏽',
    }],
  },
];

const messages = {
  groups: [
    {key: 'smileys-emotion', message: 'smileys & emotion', order: 0},
    {key: 'people-body', message: 'people & body', order: 1},
  ],
  skinTones: [
    {key: 'light', message: 'light skin tone'},
    {key: 'medium-light', message: 'medium-light skin tone'},
    {key: 'medium', message: 'medium skin tone'},
    {key: 'medium-dark', message: 'medium-dark skin tone'},
    {key: 'dark', message: 'dark skin tone'},
  ],
};

describe('EmojiPickerComponent', () => {
  let fixture: ComponentFixture<EmojiPickerComponent>;
  let component: EmojiPickerComponent;
  let testAccess: EmojiPickerTestAccess;
  let http: HttpTestingController;
  const originalDocumentLanguage = document.documentElement.lang;

  async function createComponent(locale: string): Promise<void> {
    TestBed.resetTestingModule();
    document.documentElement.lang = locale;
    await TestBed.configureTestingModule({
      imports: [EmojiPickerComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(EmojiPickerComponent);
    component = fixture.componentInstance;
    testAccess = component as unknown as EmojiPickerTestAccess;
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
  }

  afterEach(() => {
    http?.verify();
    document.documentElement.lang = originalDocumentLanguage;
    vi.restoreAllMocks();
  });

  it('loads localized data, filters by CLDR names and tags, and emits the selected skin tone', async () => {
    await createComponent('en-US');

    http.expectOne('public/emoji/en/compact.json').flush(emojiData);
    http.expectOne('public/emoji/en/messages.json').flush(messages);
    fixture.detectChanges();

    expect(component.dataLocale).toBe('en');
    expect(testAccess.groups().map(group => group.message)).toEqual(['smileys & emotion', 'people & body']);
    const formFields = fixture.nativeElement.querySelectorAll('mat-form-field') as NodeListOf<HTMLElement>;
    expect(formFields).toHaveLength(2);
    expect(Array.from(formFields).every(field => field.classList.contains('mat-form-field-appearance-outline')))
      .toBe(true);

    testAccess.searchControl.setValue('wave');
    expect(testAccess.filteredEmojis().map(emoji => emoji.unicode)).toEqual(['👋']);

    testAccess.skinToneControl.setValue('1F3FD');
    expect(testAccess.displayEmoji(emojiData[1]).unicode).toBe('👋🏽');

    const selected: string[] = [];
    component.emojiSelected.subscribe(value => selected.push(value));
    testAccess.selectEmoji(emojiData[1]);
    expect(selected).toEqual(['👋🏽']);

    const skinToneSelect = await TestbedHarnessEnvironment.loader(fixture)
      .getHarness(MatSelectHarness.with({selector: '.emoji-picker__skin-tone mat-select'}));
    await skinToneSelect.open();
    const skinToneOptions = await skinToneSelect.getOptions();
    expect(await Promise.all(skinToneOptions.map(option => option.getText())))
      .toEqual(['👋', '👋🏻', '👋🏼', '👋🏽', '👋🏾', '👋🏿']);
    const optionElements = document.querySelectorAll('mat-option') as NodeListOf<HTMLElement>;
    expect(Array.from(optionElements, option => option.getAttribute('aria-label'))).toEqual([
      'Default skin tone',
      'light skin tone',
      'medium-light skin tone',
      'medium skin tone',
      'medium-dark skin tone',
      'dark skin tone',
    ]);
    const optionTooltips = await TestbedHarnessEnvironment.documentRootLoader(fixture)
      .getAllHarnesses(MatTooltipHarness.with({selector: 'mat-option'}));
    await optionTooltips[3].show();
    expect(await optionTooltips[3].getTooltipText()).toBe('medium skin tone');
    await skinToneSelect.close();
  });

  it('uses Norwegian Bokmål assets for Norwegian locales', async () => {
    await createComponent('no');

    expect(component.dataLocale).toBe('nb');
    http.expectOne('public/emoji/nb/compact.json').flush([]);
    http.expectOne('public/emoji/nb/messages.json').flush({groups: [], skinTones: []});
  });

  it('uses the application document language instead of the browser language', async () => {
    vi.spyOn(window.navigator, 'language', 'get').mockReturnValue('nb-NO');
    await createComponent('en');

    expect(component.dataLocale).toBe('en');
    http.expectOne('public/emoji/en/compact.json').flush([]);
    http.expectOne('public/emoji/en/messages.json').flush({groups: [], skinTones: []});
  });
});
