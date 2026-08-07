import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AnnotationComponent} from './annotation.component';
import {AuthService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {Subject} from 'rxjs';

import {Annotation} from '../../../../shared/models';
import {AnnotationEditDialogResult} from './annotation-edit-dialog/annotation-edit-dialog.component';

describe('AnnotationComponent', () => {
  let component: AnnotationComponent;
  let fixture: ComponentFixture<AnnotationComponent>;
  let dialogClosed: Subject<AnnotationEditDialogResult | undefined>;
  let dialog: {open: ReturnType<typeof vi.fn>};

  beforeEach(() => {
    dialogClosed = new Subject<AnnotationEditDialogResult | undefined>();
    dialog = {
      open: vi.fn(() => ({afterClosed: () => dialogClosed.asObservable()})),
    };

    TestBed.configureTestingModule({
      imports: [
        AnnotationComponent
      ],
      providers: [
        ...provideCoreTesting,
        {
          provide: AuthService,
          useValue: {
            isAdmin: () => true,
            canUpdate: () => true,
          }
        },
        {provide: AbilityServiceSignal, useValue: {can: () => true}},
        {provide: MatDialog, useValue: dialog},
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(AnnotationComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('starts an annotation value and focuses the input from a suggestion chip', () => {
    fixture.componentRef.setInput('suggestions', ['scope_altSeeds']);
    fixture.detectChanges();

    const suggestion = fixture.nativeElement.querySelector('.annotation-suggestions mat-chip') as HTMLElement;
    const input = fixture.nativeElement.querySelector('input[placeholder="New annotation..."]') as HTMLInputElement;
    suggestion.click();

    expect(input.value).toBe('scope_altSeeds:');
    expect(document.activeElement).toBe(input);
    expect(input.selectionStart).toBe(input.value.length);
  });

  it('opens an edit dialog and applies its result to the selected annotation', () => {
    const onChange = vi.fn();
    component.registerOnChange(onChange);
    component.writeValue([
      new Annotation({key: 'scope_altSeeds', value: 'old.example'}),
      new Annotation({key: 'scope_allowedSchemes', value: 'https'}),
    ]);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('mat-chip-row') as HTMLElement).click();
    expect(dialog.open).toHaveBeenCalledWith(
      expect.any(Function),
      expect.objectContaining({
        data: {key: 'scope_altSeeds', value: 'old.example'},
        width: '480px',
        maxWidth: 'calc(100vw - 32px)',
        autoFocus: false,
        restoreFocus: true,
      })
    );

    dialogClosed.next({key: 'scope_altSeeds', value: 'new.example'});
    dialogClosed.complete();

    expect(onChange).toHaveBeenCalledWith([
      expect.objectContaining({key: 'scope_altSeeds', value: 'new.example'}),
      expect.objectContaining({key: 'scope_allowedSchemes', value: 'https'}),
    ]);
    expect(fixture.nativeElement.textContent).not.toContain('Edit annotation');
  });

  it('leaves annotations unchanged when editing is dismissed', () => {
    const onChange = vi.fn();
    component.registerOnChange(onChange);
    component.writeValue([new Annotation({key: 'scope_altSeeds', value: 'old.example'})]);

    component.onClickAnnotation('scope_altSeeds', 'old.example');
    dialogClosed.next(undefined);
    dialogClosed.complete();

    expect(onChange).not.toHaveBeenCalled();
  });

  it('removes annotations without opening the edit dialog', () => {
    const onChange = vi.fn();
    component.registerOnChange(onChange);
    component.writeValue([new Annotation({key: 'scope_altSeeds', value: 'old.example'})]);

    component.onRemoveAnnotation('scope_altSeeds', 'old.example');

    expect(onChange).toHaveBeenCalledWith([]);
    expect(dialog.open).not.toHaveBeenCalled();
  });

  it('does not open the edit dialog while disabled', () => {
    component.writeValue([new Annotation({key: 'scope_altSeeds', value: 'old.example'})]);
    component.setDisabledState(true);

    component.onClickAnnotation('scope_altSeeds', 'old.example');

    expect(dialog.open).not.toHaveBeenCalled();
  });
});
