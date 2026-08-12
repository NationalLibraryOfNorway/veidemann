import {ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Injector, Input} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR, NgControl} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';

@Component({
  selector: 'app-boolean-override',
  templateUrl: './boolean-override.component.html',
  styleUrls: ['./boolean-override.component.scss'],
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: BooleanOverrideComponent, multi: true}],
  imports: [MatButtonModule, MatChipsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class BooleanOverrideComponent implements ControlValueAccessor {
  private readonly cdr = inject(ChangeDetectorRef);
  private readonly injector = inject(Injector);

  @Input({required: true}) trueLabel = '';
  @Input({required: true}) falseLabel = '';
  @Input() ariaLabel = '';
  @Input() appearance: 'buttons' | 'chips' = 'buttons';

  value: boolean | null = null;
  disabled = false;
  private onChange: (value: boolean | null) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  select(value: boolean): void {
    if (this.disabled) return;
    this.value = this.value === value ? null : value;
    this.onChange(this.value);
    if (this.value === null) {
      this.injector.get(NgControl, null)?.control?.markAsPristine();
    }
    this.onTouched();
    this.cdr.markForCheck();
  }

  writeValue(value: boolean | null | undefined): void {
    this.value = value ?? null;
    this.cdr.markForCheck();
  }

  registerOnChange(fn: (value: boolean | null) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled = disabled;
    this.cdr.markForCheck();
  }
}
