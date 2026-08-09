import {ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR} from '@angular/forms';
import {MatChipsModule, MatChipSelectionChange} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-boolean-state-chip',
  templateUrl: './boolean-state-chip.component.html',
  styleUrls: ['./boolean-state-chip.component.scss'],
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: BooleanStateChipComponent, multi: true}],
  imports: [MatChipsModule, MatIcon, MatTooltip],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class BooleanStateChipComponent implements ControlValueAccessor {
  private readonly cdr = inject(ChangeDetectorRef);
  @Input({required: true}) label = '';
  @Input() selectedIcon = 'check_circle';
  @Input() unselectedIcon = 'cancel';
  @Input() selectedLabel = $localize`:@@commonEnabled:Enabled`;
  @Input() unselectedLabel = $localize`:@@commonDisabled:Disabled`;
  @Input() selectedTone: 'positive' | 'negative' | 'neutral' = 'positive';
  @Input() unselectedTone: 'positive' | 'negative' | 'neutral' = 'neutral';

  value = false;
  disabled = false;
  private onChange: (value: boolean) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  get ariaLabel(): string {
    return `${this.label}: ${this.value ? this.selectedLabel : this.unselectedLabel}`;
  }

  get tone(): 'positive' | 'negative' | 'neutral' {
    return this.value ? this.selectedTone : this.unselectedTone;
  }

  writeValue(value: boolean): void { this.value = !!value; this.cdr.markForCheck(); }
  registerOnChange(fn: (value: boolean) => void): void { this.onChange = fn; }
  registerOnTouched(fn: () => void): void { this.onTouched = fn; }
  setDisabledState(disabled: boolean): void { this.disabled = disabled; this.cdr.markForCheck(); }

  onSelectionChange(event: MatChipSelectionChange): void {
    if (!event.isUserInput || this.disabled) return;
    this.value = event.selected;
    this.onChange(this.value);
    this.onTouched();
  }
}
