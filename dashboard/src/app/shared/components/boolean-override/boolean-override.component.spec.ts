import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BooleanOverrideComponent} from './boolean-override.component';
import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';

describe('BooleanOverrideComponent', () => {
  let fixture: ComponentFixture<BooleanOverrideComponent>;
  let component: BooleanOverrideComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BooleanOverrideComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();
    fixture = TestBed.createComponent(BooleanOverrideComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('trueLabel', 'Enabled');
    fixture.componentRef.setInput('falseLabel', 'Disabled');
    fixture.componentRef.setInput('ariaLabel', 'State');
    fixture.detectChanges();
  });

  it('starts empty, emits both values, and clears a repeated selection', () => {
    const changes: (boolean | null)[] = [];
    component.registerOnChange(value => changes.push(value));
    const buttons = fixture.nativeElement.querySelectorAll('button') as NodeListOf<HTMLButtonElement>;

    expect(component.value).toBeNull();
    buttons[0].click();
    buttons[0].click();
    buttons[1].click();

    expect(changes).toEqual([true, null, false]);
  });

  it('renders an indicator-free single-select chip list when requested', () => {
    const changes: (boolean | null)[] = [];
    component.registerOnChange(value => changes.push(value));
    fixture.componentRef.setInput('appearance', 'chips');
    fixture.detectChanges();
    const listbox = fixture.nativeElement.querySelector('mat-chip-listbox') as HTMLElement;
    const chips = listbox.querySelectorAll('mat-chip-option') as NodeListOf<HTMLElement>;

    expect(listbox).not.toBeNull();
    expect(chips).toHaveLength(2);
    expect(listbox.querySelector('.mat-pseudo-checkbox')).toBeNull();

    chips[0].click();
    fixture.detectChanges();
    expect(chips[0].querySelector('[role="option"]')?.getAttribute('aria-selected')).toBe('true');
    expect(chips[1].querySelector('[role="option"]')?.getAttribute('aria-selected')).toBe('false');

    chips[1].click();
    fixture.detectChanges();
    expect(chips[0].querySelector('[role="option"]')?.getAttribute('aria-selected')).toBe('false');
    expect(chips[1].querySelector('[role="option"]')?.getAttribute('aria-selected')).toBe('true');
    expect(changes).toEqual([true, false]);
  });
});
