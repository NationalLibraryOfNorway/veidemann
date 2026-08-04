import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, Input, ViewChild, inject } from '@angular/core';
import {
  AbstractControl,
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  UntypedFormBuilder,
  UntypedFormControl,
  Validators
} from '@angular/forms';
import {Annotation, Label} from '../../../../shared/models';
import {COMMA, ENTER} from '@angular/cdk/keycodes';
import {MatChipInputEvent, MatChipsModule} from '@angular/material/chips';
import {NO_COLON} from '../../../../shared/validation/patterns';
import {BehaviorSubject, Observable} from 'rxjs';
import {AuthService} from '../../../../core';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {AsyncPipe} from '@angular/common';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatCardModule} from '@angular/material/card';
import {MatButtonModule} from '@angular/material/button';

interface AnnotationGroup {
  key: string;
  values: string[];
}

@Component({
  selector: 'app-annotation',
  templateUrl: './annotation.component.html',
  styleUrls: ['./annotation.component.css'],
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: AnnotationComponent, multi: true}],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class AnnotationComponent implements ControlValueAccessor {
  protected fb = inject(UntypedFormBuilder);
  protected cdr = inject(ChangeDetectorRef);
  protected authService = inject(AuthService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() removable = true;

  control = new UntypedFormControl();

  // ControlValueAccessor callback functions
  onChange: (annotations: Annotation[]) => void;
  onTouched: () => void;

  annotationForm;
  annotationInputSeparators = [ENTER, COMMA];

  disabled = false;

  protected clickedIndex = -1;
  protected annotations: Annotation[];
  protected showUpdateAnnotation = false;

  protected groupsSubject = new BehaviorSubject<AnnotationGroup[]>([]);
  groups$: Observable<AnnotationGroup[]> = this.groupsSubject.asObservable();

  @ViewChild('chipInput') chipInputControl: ElementRef;

  constructor() {
    this.createForm();
    this.can = this.abilityService.can;
  }

  get canEdit(): boolean {
    return this.authService.canUpdate('annotation');
  }

  get showUpdate(): boolean {
    return this.showUpdateAnnotation;
  }

  get canUpdate(): boolean {
    return !this.disabled;
  }

  get key(): AbstractControl {
    return this.annotationForm.get('key');
  }

  get value(): AbstractControl {
    return this.annotationForm.get('value');
  }

  writeValue(annotations: Annotation[]): void {
    if (annotations === null) {
      this.annotations = [];
    } else {
      this.annotations = annotations.map(annotation => new Annotation({key: annotation.key, value: annotation.value}));
    }
    this.reset();
  }

  // implement ControlValueAccessor
  registerOnChange(fn: (annotations: Annotation[]) => void): void {
    this.onChange = fn;
  }

  // implement ControlValueAccessor
  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  // implement ControlValueAccessor
  setDisabledState(disabled: boolean): void {
    this.disabled = disabled;
    if (this.disabled) {
      this.control.disable();
    } else {
      this.control.enable();
    }
    this.cdr.markForCheck();
  }

  onClickAnnotation(key: string, value: string): void {
    if (this.disabled) {
      return;
    }
    this.showUpdateAnnotation = true;
    this.clickedIndex = this.findAnnotationIndex(key, value);
    this.annotationForm.enable();
    this.annotationForm.reset({key, value});
  }

  onSave(event: MatChipInputEvent): void {
    this.save(event.value);
    this.onChange(this.annotations);
    this.reset();

    this.chipInputControl.nativeElement.value = '';
  }

  protected save(value: string): void {
    let key = '';
    value = value.trim();

    if (value === '') {
      return;
    }

    const parts = value.split(':');
    if (parts.length > 1) {
      key = parts.shift();
      value = parts.join(':');
      if (key.length < 1 || value.length < 1) {
        return;
      }
    } else {
      return;
    }

    if (this.findAnnotationIndex(key, value) > -1) {
      return;
    }

    this.annotations.push(new Label({key, value}));
  }

  onUpdateAnnotation(key: string, value: string): void {
    key = key.trim();
    value = value.trim();

    // remove old
    this.annotations.splice(this.clickedIndex, 1);
    // add updated
    this.annotations.push(new Annotation({key, value}));

    this.onChange(this.annotations);
    this.reset();
  }

  onRemoveAnnotation(key: string, value: string): void {
    const index = this.findAnnotationIndex(key, value);
    if (index !== -1) {
      this.annotations.splice(index, 1);
    }
    this.onChange(this.annotations);
    this.reset();
  }

  onAbort(): void {
    this.annotationForm.disable();
  }

  protected reset() {
    this.regroup();
    this.annotationForm.reset();
    this.annotationForm.disable();
    this.cdr.detectChanges();
  }

  protected findAnnotationIndex(key: string, value: string): number {
    return this.annotations.findIndex((element) => {
      return element.key === key && element.value === value;
    });
  }

  protected createForm(): void {
    this.annotationForm = this.fb.group({
      key: ['', [Validators.required, Validators.pattern(NO_COLON)]],
      value: ['', Validators.required]
    });
    this.annotationForm.disable();
    if (!this.canEdit) {
      this.setDisabledState(true);
    }

  }

  // group annotations with similar key together
  protected regroup(): void {
    const grouping: Record<string, string[]> = {};

    this.annotations.forEach(annotation => {
      if (Object.hasOwn(grouping, annotation.key)) {
        grouping[annotation.key].push(annotation.value);
      } else {
        grouping[annotation.key] = [annotation.value];
      }
    });
    this.groupsSubject.next(Object.keys(grouping).map(key => ({key, values: grouping[key]})));
  }


}
