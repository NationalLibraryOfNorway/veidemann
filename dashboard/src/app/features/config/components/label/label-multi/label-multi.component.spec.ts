import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelMultiComponent} from './label-multi.component';
import {ConfigObject, Label} from '../../../../../shared/models/config';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatDialog} from '@angular/material/dialog';
import {of} from 'rxjs';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatChipListboxHarness} from '@angular/material/chips/testing';

describe('LabelMultiComponent', () => {
  let component: LabelMultiComponent;
  let fixture: ComponentFixture<LabelMultiComponent>;
  let loader: HarnessLoader;
  let dialog: {open: ReturnType<typeof vi.fn>};

  beforeEach(() => {
    dialog = {open: vi.fn(() => ({afterClosed: () => of('🐶')}))};
    TestBed.configureTestingModule({
      imports: [LabelMultiComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialog, useValue: dialog},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelMultiComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject();
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

  it('offers the picker only in add mode and emits a selected emoji once', async () => {
    const updates = [];
    component.update.subscribe(update => updates.push({add: update.add, labels: [...update.labels]}));

    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).toBeNull();
    const toggleButtons = fixture.nativeElement.querySelectorAll('mat-button-toggle button') as NodeListOf<HTMLButtonElement>;
    toggleButtons[0].click();
    fixture.detectChanges();
    expect(component.shouldAddLabel).toBe(true);
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).not.toBeNull();

    await component.onChooseEmoji();
    await component.onChooseEmoji();

    expect(dialog.open).toHaveBeenCalledTimes(2);
    expect(updates.at(-1)).toEqual({add: true, labels: [{key: 'emoji', value: '🐶'}]});
    expect(updates.filter(update => update.labels.length > 0)).toHaveLength(1);

    toggleButtons[1].click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).toBeNull();
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

  it('allocates layout height for the visible label editor', () => {
    selectMode(0);

    const editor = fixture.nativeElement.querySelector('.label-input-section') as HTMLElement;
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
