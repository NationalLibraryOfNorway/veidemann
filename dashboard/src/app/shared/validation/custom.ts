import {AbstractControl, UntypedFormControl, ValidationErrors, ValidatorFn} from '@angular/forms';

export class CustomValidators {
  /**
   * Validator that requires controls to have a value greater than a number.
   */
  static max(max: number): ValidatorFn {
    return (control: UntypedFormControl): Record<string, boolean> | null => {

      const val: number = control.value;

      if (control.pristine || control.pristine) {
        return null;
      }
      if (val <= max) {
        return null;
      }
      return {max: true};
    };
  }

  static min(min: number): ValidatorFn {
    return (control: UntypedFormControl): Record<string, boolean> | null => {

      const val: number = control.value;

      if (control.pristine || control.pristine) {
        return null;
      }
      if (val >= min) {
        return null;
      }
      return {min: true};
    };
  }

  static nonEmpty(control: AbstractControl): ValidationErrors | null {
    if (!control.value || control.value.length === 0) {
      return {nonEmpty: true};
    } else {
      return null;
    }

  }
}
