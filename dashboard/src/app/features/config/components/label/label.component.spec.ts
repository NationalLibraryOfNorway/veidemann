import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelComponent} from './label.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {MatDialog} from '@angular/material/dialog';
import {firstValueFrom, of} from 'rxjs';
import {skip} from 'rxjs/operators';
import {Kind, Label} from '../../../../shared/models';
import {LabelService} from '../../services/label.service';

describe('LabelsComponent', () => {
  let component: LabelComponent;
  let fixture: ComponentFixture<LabelComponent>;
  let dialogResult: unknown;
  let dialog: {open: ReturnType<typeof vi.fn>};
  let getLabelKeys: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    dialogResult = '🐶';
    dialog = {open: vi.fn(() => ({afterClosed: () => of(dialogResult)}))};
    getLabelKeys = vi.fn(() => of(['owner', 'category']));
    TestBed.configureTestingModule({
      imports: [LabelComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialog, useValue: dialog},
        {provide: LabelService, useValue: {getLabelKeys}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('kind', Kind.CRAWLENTITY);
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('requests entity label keys and filters them case-insensitively', async () => {
    const filteredKeys = firstValueFrom(component.filteredKey$.pipe(skip(1)));
    component.control.setValue('OW');

    await expect(filteredKeys).resolves.toEqual(['owner']);
    expect(getLabelKeys).toHaveBeenCalledWith(Kind.CRAWLENTITY);
  });

  it('adds a selected emoji label once through the control value accessor', async () => {
    const changes: Label[][] = [];
    component.registerOnChange(labels => changes.push([...labels]));
    component.writeValue([]);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).not.toBeNull();
    await component.onChooseEmoji();
    await component.onChooseEmoji();

    expect(dialog.open).toHaveBeenCalledTimes(2);
    expect(changes).toEqual([[new Label({key: 'emoji', value: '🐶'})]]);
    expect(fixture.nativeElement.querySelector('.label-display__emoji').textContent).toBe('🐶');
  });

  it('hides the picker while disabled and ignores cancelled selection', async () => {
    const changes = vi.fn();
    component.registerOnChange(changes);
    component.writeValue([]);
    dialogResult = undefined;
    component.setDisabledState(true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).toBeNull();
    await component.onChooseEmoji();
    expect(dialog.open).not.toHaveBeenCalled();
    expect(changes).not.toHaveBeenCalled();
  });

  it('opens a focused label dialog and applies only its confirmed result', () => {
    const changes: Label[][] = [];
    component.registerOnChange(labels => changes.push([...labels]));
    component.writeValue([new Label({key: 'category', value: 'old'})]);
    dialogResult = {key: 'category', value: 'new'};
    fixture.detectChanges();

    component.onClickLabel('category', 'old');

    expect(dialog.open).toHaveBeenCalledWith(expect.any(Function), expect.objectContaining({
      data: {key: 'category', value: 'old', type: 'label'},
    }));
    expect(changes).toEqual([[new Label({key: 'category', value: 'new'})]]);
    expect(fixture.nativeElement.textContent).not.toContain('Edit label');
  });

  it('removes a label without also opening its edit dialog', () => {
    const changes: Label[][] = [];
    component.registerOnChange(labels => changes.push([...labels]));
    component.writeValue([new Label({key: 'category', value: 'news'})]);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('button[matChipRemove]') as HTMLButtonElement).click();

    expect(dialog.open).not.toHaveBeenCalled();
    expect(changes).toEqual([[]]);
  });

  it('copies a label dropped from a related configuration card', () => {
    const changes: Label[][] = [];
    component.registerOnChange(labels => changes.push([...labels]));
    component.writeValue([]);
    const preventDefault = vi.fn();

    component.onNativeDrop({
      preventDefault,
      dataTransfer: {getData: () => 'category:news'},
    } as unknown as DragEvent);

    expect(preventDefault).toHaveBeenCalled();
    expect(changes).toEqual([[new Label({key: 'category', value: 'news'})]]);
  });
});
