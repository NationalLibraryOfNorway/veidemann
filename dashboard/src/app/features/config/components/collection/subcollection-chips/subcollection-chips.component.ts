import {ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnDestroy} from '@angular/core';
import {
  AbstractControl,
  ControlValueAccessor,
  NG_VALIDATORS,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  UntypedFormArray,
  UntypedFormBuilder,
  UntypedFormGroup,
  ValidationErrors,
  Validator,
  Validators,
} from '@angular/forms';
import {MatChipsModule} from '@angular/material/chips';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatSelectModule} from '@angular/material/select';
import {Subscription} from 'rxjs';
import {SubCollection, SubCollectionType} from '../../../../../shared/models';
import {VALID_COLLECTION_NAME} from '../../../../../shared/validation/patterns';

@Component({
  selector: 'app-subcollection-chips',
  template: `
    <fieldset class="subcollection-group">
      <legend i18n="@@collectionFormSubCollectionTitle">Subcollections</legend>
      <div [formGroup]="form">
        <div formArrayName="items">
          @for (item of itemControls.controls; track item; let index = $index) {
            <div [formGroupName]="index" class="subcollection-row layout-row gap-16"
              data-testid="subcollectionRow">
              <mat-form-field class="flex-fill">
                <mat-label i18n="@@collectionFormSubCollectionNameLabel">Name</mat-label>
                <input matInput formControlName="name" data-testid="subcollectionName">
                @if (shouldShowNameError(index)) {
                  <mat-error i18n="@@collectionMetaSubcollectionNamePatternError">
                    Enter at least two valid characters.
                  </mat-error>
                }
              </mat-form-field>
              <mat-form-field class="flex-fill">
                <mat-label i18n="@@collectionSubCollectionTypeDropdownLabel">Subcollection type</mat-label>
                <mat-select formControlName="type" data-testid="subcollectionType">
                  <mat-option [value]="typeControl(index).value">
                    {{typeLabel(typeControl(index).value)}}
                  </mat-option>
                </mat-select>
              </mat-form-field>
              @if (canInteract) {
                <button mat-icon-button type="button" (click)="remove(index)"
                  [attr.aria-label]="removeLabel(index)" data-testid="removeSubcollectionButton">
                  <mat-icon>remove_circle</mat-icon>
                </button>
              }
            </div>
          }
        </div>
      </div>

      @if (hasDuplicateTypes) {
        <mat-error i18n="@@collectionDuplicateSubcollectionTypeError" data-testid="duplicateSubcollectionTypeError">
          Only one subcollection of each type is allowed.
        </mat-error>
      }

      @if (canInteract) {
        <mat-chip-set class="subcollection-add" aria-label="Subcollection actions"
          i18n-aria-label="@@collectionSubcollectionActionsLabel">
          @if (supportsType(SubCollectionType.SCREENSHOT) && !hasType(SubCollectionType.SCREENSHOT)) {
            <mat-chip role="button" tabindex="0" (click)="add(SubCollectionType.SCREENSHOT)"
              (keydown)="addKeydown($event, SubCollectionType.SCREENSHOT)"
              data-testid="addScreenshotSubcollectionButton">
              <mat-icon matChipAvatar>add_circle</mat-icon>
              <span i18n="@@collectionAddScreenshotSubcollectionButton">Add screenshot collection</span>
            </mat-chip>
          }
          @if (supportsType(SubCollectionType.DNS) && !hasType(SubCollectionType.DNS)) {
            <mat-chip role="button" tabindex="0" (click)="add(SubCollectionType.DNS)"
              (keydown)="addKeydown($event, SubCollectionType.DNS)"
              data-testid="addDnsSubcollectionButton">
              <mat-icon matChipAvatar>add_circle</mat-icon>
              <span i18n="@@collectionAddDnsSubcollectionButton">Add DNS collection</span>
            </mat-chip>
          }
        </mat-chip-set>
      }
    </fieldset>
  `,
  styles: [`
    .subcollection-group { border: 0; margin: 24px 0 0; padding: 0; }
    .subcollection-group legend { margin-bottom: 16px; font: var(--mat-sys-title-small); }
    .subcollection-row { align-items: start; }
    .subcollection-row button { margin-top: 8px; }
    .subcollection-add { display: block; margin-top: 8px; }
    .subcollection-add mat-chip[role='button'] { cursor: pointer; }
    .subcollection-add mat-chip[role='button'] * { cursor: pointer; }
  `],
  providers: [
    {provide: NG_VALUE_ACCESSOR, useExisting: SubcollectionChipsComponent, multi: true},
    {provide: NG_VALIDATORS, useExisting: SubcollectionChipsComponent, multi: true},
  ],
  imports: [
    MatChipsModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    MatSelectModule,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class SubcollectionChipsComponent implements ControlValueAccessor, OnDestroy, Validator {
  private readonly fb = inject(UntypedFormBuilder);
  private readonly cdr = inject(ChangeDetectorRef);
  private readonly subscription: Subscription;
  private disabled = false;
  private isEditable = true;
  private writingValue = false;
  private onChange: (value: SubCollection[]) => void = () => undefined;
  private onTouched: () => void = () => undefined;
  private onValidatorChange: () => void = () => undefined;

  readonly SubCollectionType = SubCollectionType;
  readonly form = this.fb.group({items: this.fb.array([])});

  @Input() types: SubCollectionType[] = [];
  @Input()
  set editable(editable: boolean) {
    this.isEditable = editable;
    this.updateDisabledState();
  }

  constructor() {
    this.subscription = this.itemControls.valueChanges.subscribe(() => {
      if (this.writingValue) return;
      this.onChange(this.subcollections);
      this.onTouched();
      this.onValidatorChange();
      this.cdr.markForCheck();
    });
  }

  get itemControls(): UntypedFormArray {
    return this.form.get('items') as UntypedFormArray;
  }

  get canInteract(): boolean {
    return this.isEditable && !this.disabled;
  }

  get hasDuplicateTypes(): boolean {
    const types = this.subcollections.map(item => item.type);
    return new Set(types).size !== types.length;
  }

  private get subcollections(): SubCollection[] {
    return this.itemControls.getRawValue().map(value => new SubCollection(value));
  }

  writeValue(value: SubCollection[] | null): void {
    this.writingValue = true;
    this.itemControls.clear({emitEvent: false});
    for (const item of value ?? []) {
      this.itemControls.push(this.createItem(item), {emitEvent: false});
    }
    this.updateDisabledState();
    this.writingValue = false;
    this.onValidatorChange();
    this.cdr.markForCheck();
  }

  registerOnChange(fn: (value: SubCollection[]) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  registerOnValidatorChange(fn: () => void): void {
    this.onValidatorChange = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled = disabled;
    this.updateDisabledState();
    this.cdr.markForCheck();
  }

  private updateDisabledState(): void {
    if (!this.canInteract) {
      this.form.disable({emitEvent: false});
    } else {
      this.form.enable({emitEvent: false});
      this.itemControls.controls.forEach(control => control.get('type')?.disable({emitEvent: false}));
    }
  }

  validate(): ValidationErrors | null {
    const errors: ValidationErrors = {};
    if (this.itemControls.invalid) errors['invalidSubcollections'] = true;
    if (this.hasDuplicateTypes) errors['duplicateSubcollectionTypes'] = true;
    return Object.keys(errors).length ? errors : null;
  }

  add(type: SubCollectionType): void {
    if (!this.canInteract || !this.supportsType(type) || this.hasType(type)) return;
    this.itemControls.push(this.createItem(new SubCollection({type})));
  }

  addKeydown(event: KeyboardEvent, type: SubCollectionType): void {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    this.add(type);
  }

  remove(index: number): void {
    if (!this.canInteract) return;
    this.itemControls.removeAt(index);
  }

  hasType(type: SubCollectionType): boolean {
    return this.subcollections.some(item => item.type === type);
  }

  supportsType(type: SubCollectionType): boolean {
    const types = this.types?.length
      ? this.types
      : [SubCollectionType.SCREENSHOT, SubCollectionType.DNS];
    return types.includes(type);
  }

  nameControl(index: number): AbstractControl {
    return this.itemControls.at(index).get('name') as AbstractControl;
  }

  typeControl(index: number): AbstractControl {
    return this.itemControls.at(index).get('type') as AbstractControl;
  }

  shouldShowNameError(index: number): boolean {
    const control = this.nameControl(index);
    return control.invalid && (control.dirty || control.touched);
  }

  typeLabel(type: SubCollectionType): string {
    return type === SubCollectionType.SCREENSHOT
      ? $localize`:@@subcollectionTypeScreenshot:Screenshot`
      : $localize`:@@subcollectionTypeDns:DNS`;
  }

  removeLabel(index: number): string {
    return $localize`:@@removeSubcollectionAriaLabel:Remove ${this.nameControl(index).value}:SUBCOLLECTION_NAME: subcollection`;
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  private createItem(item: SubCollection): UntypedFormGroup {
    return this.fb.group({
      name: [item.name, [Validators.required, Validators.minLength(2), Validators.pattern(VALID_COLLECTION_NAME)]],
      type: [{value: item.type, disabled: true}],
    });
  }
}
