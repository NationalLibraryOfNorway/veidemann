import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelComponent} from './label.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {MatDialog} from '@angular/material/dialog';
import {of} from 'rxjs';
import {Label} from '../../../../shared/models';

describe('LabelsComponent', () => {
  let component: LabelComponent;
  let fixture: ComponentFixture<LabelComponent>;
  let dialogResult: string | undefined;
  let dialog: {open: ReturnType<typeof vi.fn>};

  beforeEach(() => {
    dialogResult = '🐶';
    dialog = {open: vi.fn(() => ({afterClosed: () => of(dialogResult)}))};
    TestBed.configureTestingModule({
      imports: [LabelComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialog, useValue: dialog},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
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
});
