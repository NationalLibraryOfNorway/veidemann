import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelMultiComponent} from './label-multi.component';
import {ConfigObject, Kind, Label} from '../../../../../shared/models/config';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatDialog} from '@angular/material/dialog';
import {firstValueFrom, of} from 'rxjs';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {LabelService} from '../../../services/label.service';

describe('LabelMultiComponent', () => {
  let component: LabelMultiComponent;
  let fixture: ComponentFixture<LabelMultiComponent>;
  let loader: HarnessLoader;
  let dialog: {open: ReturnType<typeof vi.fn>};
  let getLabelKeys: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    dialog = {open: vi.fn(() => ({afterClosed: () => of('🐶')}))};
    getLabelKeys = vi.fn(() => of(['owner']));
    TestBed.configureTestingModule({
      imports: [LabelMultiComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialog, useValue: dialog},
        {provide: LabelService, useValue: {getLabelKeys}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelMultiComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLENTITY});
    component.configObject.meta.labelList = [
      new Label({key: 'type', value: 'default'}),
      new Label({key: 'category', value: 'news'}),
    ];
    loader = TestbedHarnessEnvironment.loader(fixture);
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('derives the label-key request kind from the bulk-edit configuration', async () => {
    await firstValueFrom(component.filteredKey$);

    expect(getLabelKeys).toHaveBeenCalledWith(Kind.CRAWLENTITY);
  });

  it('keeps the disabled editor visible and offers the picker in both modes', async () => {
    const updates = [];
    component.update.subscribe(update => updates.push({add: update.add, labels: [...update.labels]}));

    const initialEmojiButton = fixture.nativeElement.querySelector(
      '[data-testid="emoji-picker-button"]'
    ) as HTMLButtonElement;
    expect(initialEmojiButton).not.toBeNull();
    expect(initialEmojiButton.disabled).toBe(true);
    expect(component.control.disabled).toBe(true);
    const toggleButtons = fixture.nativeElement.querySelectorAll('mat-button-toggle button') as NodeListOf<HTMLButtonElement>;
    toggleButtons[0].click();
    fixture.detectChanges();
    expect(component.shouldAddLabel).toBe(true);
    expect(fixture.nativeElement.querySelector('mat-hint')?.textContent.trim())
      .toBe('Type a label or click a common label to add it to the input.');
    const emojiButton = fixture.nativeElement.querySelector(
      '[data-testid="emoji-picker-button"]'
    ) as HTMLButtonElement;
    expect(emojiButton).not.toBeNull();
    expect(emojiButton.disabled).toBe(false);
    expect(component.control.enabled).toBe(true);
    expect(emojiButton.closest('mat-form-field')).not.toBeNull();

    await component.onChooseEmoji();
    await component.onChooseEmoji();

    expect(dialog.open).toHaveBeenCalledTimes(2);
    expect(updates.at(-1)).toEqual({add: true, labels: [{key: 'emoji', value: '🐶'}]});
    expect(updates.filter(update => update.labels.length > 0)).toHaveLength(1);

    toggleButtons[1].click();
    fixture.detectChanges();
    expect(component.shouldAddLabel).toBe(false);
    expect(fixture.nativeElement.querySelector('mat-hint')?.textContent.trim())
      .toBe('Type a label or click a common label to add it to the input.');
    expect((fixture.nativeElement.querySelector(
      '[data-testid="emoji-picker-button"]'
    ) as HTMLButtonElement).disabled).toBe(false);

    await component.onChooseEmoji();
    expect(updates.at(-1)).toEqual({add: false, labels: [{key: 'emoji', value: '🐶'}]});
  });

  it('keeps common labels inactive until an update mode is selected', async () => {
    const listbox = await getCommonLabelListbox();
    const [label] = await listbox.getChips();

    expect(await listbox.isMultiple()).toBe(true);
    await label.select();

    expect(await label.isSelected()).toBe(false);
    expect(component.labels).toEqual([]);
  });

  it('adds and removes multiple common labels in add mode', async () => {
    const updates = [];
    component.update.subscribe(update => updates.push({add: update.add, labels: [...update.labels]}));
    selectMode(0);

    const listbox = await getCommonLabelListbox();
    const [first, second] = await listbox.getChips();
    await first.select();
    await second.select();

    expect(component.labels).toEqual([
      {key: 'type', value: 'default'},
      {key: 'category', value: 'news'},
    ]);
    expect(updates.at(-1).add).toBe(true);

    await first.deselect();
    expect(component.labels).toEqual([{key: 'category', value: 'news'}]);
  });

  it('adds a common label to the pending input in remove mode', async () => {
    const updates = [];
    component.update.subscribe(update => updates.push({add: update.add, labels: [...update.labels]}));
    selectMode(1);

    const listbox = await getCommonLabelListbox();
    const [first] = await listbox.getChips();
    await first.select();

    expect(component.labels).toEqual([{key: 'type', value: 'default'}]);
    expect(updates.at(-1)).toEqual({add: false, labels: [{key: 'type', value: 'default'}]});
  });

  it('keeps common-label selection synchronized with input removal and prevents duplicates', async () => {
    selectMode(0);
    const listbox = await getCommonLabelListbox();
    const [first] = await listbox.getChips();
    await first.select();

    component.onAdd({input: undefined, chipInput: null, value: 'type:default'});
    expect(component.labels).toHaveLength(1);

    component.onRemove('type', 'default');
    fixture.detectChanges();
    expect(await first.isSelected()).toBe(false);
  });

  it('clears pending common labels when switching mode or reverting', async () => {
    selectMode(0);
    const listbox = await getCommonLabelListbox();
    const [first] = await listbox.getChips();
    await first.select();

    selectMode(1);
    expect(component.labels).toEqual([]);
    expect(await first.isSelected()).toBe(false);

    await first.select();
    component.onRevert();
    fixture.detectChanges();
    expect(component.labels).toEqual([]);
    expect(await first.isSelected()).toBe(false);
  });

  it('right-aligns the indicator-free toggle and uses a two-column value row', () => {
    const headingRow = fixture.nativeElement.querySelector('.label-heading-row') as HTMLElement;
    const valueRow = fixture.nativeElement.querySelector('.label-value-row') as HTMLElement;
    const heading = headingRow.querySelector('h5') as HTMLElement;
    const toggle = headingRow.querySelector('mat-button-toggle-group') as HTMLElement;
    const commonLabels = valueRow.querySelector('#commonLabelsChipList') as HTMLElement;
    const editor = fixture.nativeElement.querySelector('.label-input-section') as HTMLElement;

    expect(heading.nextElementSibling).toBe(toggle);
    expect(toggle.getBoundingClientRect().right).toBeCloseTo(headingRow.getBoundingClientRect().right, 0);
    expect(toggle.querySelector('.mat-pseudo-checkbox')).toBeNull();
    expect(commonLabels.parentElement).toBe(valueRow);
    expect(editor.parentElement).toBe(valueRow);
    expect(getComputedStyle(valueRow).display).toBe('grid');
    expect(getComputedStyle(valueRow).gridTemplateColumns).toBe('repeat(2, minmax(0, 1fr))');
    expect(editor).not.toBeNull();
    expect(getComputedStyle(editor).display).toBe('block');
    expect(editor.classList.contains('flex-fill')).toBe(false);
  });

  function selectMode(index: number): void {
    const buttons = fixture.nativeElement.querySelectorAll('mat-button-toggle button') as NodeListOf<HTMLButtonElement>;
    buttons[index].click();
    fixture.detectChanges();
  }

  function getCommonLabelListbox(): Promise<MatChipListboxHarness> {
    return loader.getHarness(MatChipListboxHarness.with({selector: '#commonLabelsChipList'}));
  }
});
