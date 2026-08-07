import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {
  AnnotationEditDialogComponent,
  AnnotationEditDialogData,
  AnnotationEditDialogResult,
} from './annotation-edit-dialog.component';

describe('AnnotationEditDialogComponent', () => {
  let fixture: ComponentFixture<AnnotationEditDialogComponent>;
  let component: AnnotationEditDialogComponent;
  let dialogRef: {close: ReturnType<typeof vi.fn>};

  const data: AnnotationEditDialogData = {
    key: 'scope_altSeeds',
    value: 'old.example',
  };

  beforeEach(async () => {
    dialogRef = {close: vi.fn()};

    await TestBed.configureTestingModule({
      imports: [AnnotationEditDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: dialogRef},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AnnotationEditDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('initializes the form and selects the value field', () => {
    const valueInput = fixture.nativeElement.querySelector('input[formcontrolname="value"]') as HTMLInputElement;
    const applyButton = fixture.nativeElement.querySelector('button[type="submit"]') as HTMLButtonElement;

    expect(component.form.getRawValue()).toEqual(data);
    expect(document.activeElement).toBe(valueInput);
    expect(valueInput.selectionStart).toBe(0);
    expect(valueInput.selectionEnd).toBe(data.value.length);
    expect(component.canApply).toBe(false);
    expect(applyButton.disabled).toBe(true);
  });

  it('rejects empty, whitespace-only, and colon-containing values', () => {
    component.key.setValue('scope:altSeeds');
    expect(component.key.invalid).toBe(true);

    component.key.setValue('   ');
    component.value.setValue('   ');
    expect(component.form.invalid).toBe(true);
  });

  it('returns trimmed values only after a valid change', () => {
    component.key.setValue(' scope_altSeeds ');
    component.value.setValue(' new.example ');
    component.form.markAsDirty();
    fixture.detectChanges();
    expect(component.canApply).toBe(true);
    const applyButton = fixture.nativeElement.querySelector('button[type="submit"]') as HTMLButtonElement;
    expect(applyButton.disabled).toBe(false);

    component.onApply();

    expect(dialogRef.close).toHaveBeenCalledWith({
      key: 'scope_altSeeds',
      value: 'new.example',
    } satisfies AnnotationEditDialogResult);
  });

  it('does not close with an unchanged or invalid form', () => {
    component.form.markAsDirty();
    component.onApply();
    component.value.setValue('   ');
    component.onApply();

    expect(dialogRef.close).not.toHaveBeenCalled();
  });
});
