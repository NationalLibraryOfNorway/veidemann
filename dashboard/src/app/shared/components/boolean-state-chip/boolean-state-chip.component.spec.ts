import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatChipSelectionChange} from '@angular/material/chips';

import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {BooleanStateChipComponent} from './boolean-state-chip.component';

describe('BooleanStateChipComponent', () => {
  let fixture: ComponentFixture<BooleanStateChipComponent>;
  let component: BooleanStateChipComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BooleanStateChipComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();
    fixture = TestBed.createComponent(BooleanStateChipComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('label', 'Compression');
  });

  it('renders selected and unselected icons, tones, and accessible labels', () => {
    component.writeValue(false);
    fixture.detectChanges();
    let chip = fixture.nativeElement.querySelector('mat-chip-option') as HTMLElement;
    expect(chip.textContent).toContain('Disabled');
    expect(chip.textContent).not.toContain('Compression:');
    expect(chip.textContent).toContain('cancel');
    expect(fixture.nativeElement.querySelector('[aria-label="Compression: Disabled"]')).not.toBeNull();
    expect(chip.classList.contains('state-selected')).toBe(false);

    component.writeValue(true);
    fixture.detectChanges();
    chip = fixture.nativeElement.querySelector('mat-chip-option') as HTMLElement;
    expect(chip.textContent).toContain('Enabled');
    expect(chip.textContent).not.toContain('Compression:');
    expect(chip.textContent).toContain('check_circle');
    expect(fixture.nativeElement.querySelector('[aria-label="Compression: Enabled"]')).not.toBeNull();
    expect(chip.classList.contains('state-selected')).toBe(true);
  });

  it('propagates only user-initiated selection changes', () => {
    const change = vi.fn();
    component.registerOnChange(change);
    component.onSelectionChange({isUserInput: false, selected: true} as MatChipSelectionChange);
    component.onSelectionChange({isUserInput: true, selected: true} as MatChipSelectionChange);
    expect(change).toHaveBeenCalledOnce();
    expect(change).toHaveBeenCalledWith(true);
  });
});
