import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AnnotationComponent} from './annotation.component';
import {AuthService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {Subject} from 'rxjs';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatMenuHarness} from '@angular/material/menu/testing';

import {Annotation} from '../../../../shared/models';
import {AnnotationEditDialogResult} from './annotation-edit-dialog/annotation-edit-dialog.component';

describe('AnnotationComponent', () => {
  let component: AnnotationComponent;
  let fixture: ComponentFixture<AnnotationComponent>;
  let dialogClosed: Subject<AnnotationEditDialogResult | undefined>;
  let dialog: {open: ReturnType<typeof vi.fn>};
  let loader: HarnessLoader;

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
    loader = TestbedHarnessEnvironment.loader(fixture);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('collapses annotation key suggestions behind one counted trigger chip', async () => {
    fixture.componentRef.setInput('suggestions', ['scope_altSeeds', 'scope_maxHopsFromSeed']);
    fixture.detectChanges();

    const triggers = fixture.nativeElement.querySelectorAll('.annotation-suggestions mat-chip') as NodeListOf<HTMLElement>;
    const trigger = triggers[0];
    expect(triggers).toHaveLength(1);
    expect(trigger.textContent.replace(/\s+/g, ' ').trim()).toContain('Suggested keys (2)');
    expect(trigger.getAttribute('aria-label')).toBe('Show suggested annotation keys');

    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();

    const items = document.querySelectorAll('.mat-mdc-menu-item') as NodeListOf<HTMLElement>;
    expect(Array.from(items).map(item => item.textContent.trim()))
      .toEqual(['scope_altSeeds', 'scope_maxHopsFromSeed']);
  });

  it('starts an annotation value from the suggestion menu without saving it', async () => {
    const onChange = vi.fn();
    component.registerOnChange(onChange);
    fixture.componentRef.setInput('suggestions', ['scope_altSeeds']);
    fixture.detectChanges();

    const trigger = fixture.nativeElement.querySelector('.annotation-suggestions__trigger') as HTMLElement;
    const input = fixture.nativeElement.querySelector('input[placeholder="New annotation..."]') as HTMLInputElement;
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    (document.querySelector('.mat-mdc-menu-item') as HTMLElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(input.value).toBe('scope_altSeeds:');
    expect(document.activeElement).toBe(input);
    expect(input.selectionStart).toBe(input.value.length);
    expect(onChange).not.toHaveBeenCalled();
  });

  it.each(['Enter', ' ', 'ArrowDown'])('opens the suggestion menu with %s', async key => {
    fixture.componentRef.setInput('suggestions', ['scope_altSeeds']);
    fixture.detectChanges();

    const trigger = fixture.nativeElement.querySelector('.annotation-suggestions__trigger') as HTMLElement;
    trigger.dispatchEvent(new KeyboardEvent('keydown', {key, bubbles: true, cancelable: true}));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(document.querySelector('.mat-mdc-menu-item')).not.toBeNull();
  });

  it('returns focus to the trigger when the suggestion menu is dismissed', async () => {
    fixture.componentRef.setInput('suggestions', ['scope_altSeeds']);
    fixture.detectChanges();

    const trigger = fixture.nativeElement.querySelector('.annotation-suggestions__trigger') as HTMLElement;
    const input = fixture.nativeElement.querySelector('input[placeholder="New annotation..."]') as HTMLInputElement;
    const menu = await loader.getHarness(MatMenuHarness);
    await menu.open();
    expect(await menu.isOpen()).toBe(true);
    await menu.close();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(await menu.isOpen()).toBe(false);
    expect(document.activeElement).toBe(trigger);
    expect(input.value).toBe('');
  });

  it('hides the suggestion trigger when suggestions are unavailable or editing is disabled', () => {
    expect(fixture.nativeElement.querySelector('.annotation-suggestions__trigger')).toBeNull();

    fixture.componentRef.setInput('suggestions', ['scope_altSeeds']);
    component.setDisabledState(true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.annotation-suggestions__trigger')).toBeNull();
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
