import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, Input, ViewChild, inject } from '@angular/core';
import {
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  UntypedFormControl
} from '@angular/forms';
import {Annotation, Label} from '../../../../shared/models';
import {COMMA, ENTER} from '@angular/cdk/keycodes';
import {MatChipInputEvent, MatChipsModule} from '@angular/material/chips';
import {BehaviorSubject, Observable} from 'rxjs';
import {filter, take} from 'rxjs/operators';
import {AuthService} from '../../../../core';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {AsyncPipe} from '@angular/common';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatDialog} from '@angular/material/dialog';
import {
  AnnotationEditDialogComponent,
  AnnotationEditDialogData,
  AnnotationEditDialogResult,
} from './annotation-edit-dialog/annotation-edit-dialog.component';

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
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class AnnotationComponent implements ControlValueAccessor {
  protected cdr = inject(ChangeDetectorRef);
  protected authService = inject(AuthService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private dialog = inject(MatDialog);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() removable = true;

  /** Product-authored annotation keys that can be inserted into the input. */
  @Input() suggestions: string[] = [];

  control = new UntypedFormControl();

  // ControlValueAccessor callback functions
  onChange: (annotations: Annotation[]) => void;
  onTouched: () => void;

  annotationInputSeparators = [ENTER, COMMA];

  disabled = false;

  protected annotations: Annotation[];

  protected groupsSubject = new BehaviorSubject<AnnotationGroup[]>([]);
  groups$: Observable<AnnotationGroup[]> = this.groupsSubject.asObservable();

  @ViewChild('chipInput') chipInputControl: ElementRef;

  constructor() {
    this.can = this.abilityService.can;
    if (!this.canEdit) {
      this.setDisabledState(true);
    }
  }

  get canEdit(): boolean {
    return this.authService.canUpdate('annotation');
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
    const clickedIndex = this.findAnnotationIndex(key, value);
    if (clickedIndex === -1) {
      return;
    }

    this.dialog.open<AnnotationEditDialogComponent, AnnotationEditDialogData, AnnotationEditDialogResult>(
      AnnotationEditDialogComponent,
      {
        data: {key, value},
        width: '480px',
        maxWidth: 'calc(100vw - 32px)',
        autoFocus: false,
        restoreFocus: true,
      }
    ).afterClosed().pipe(
      filter((result): result is AnnotationEditDialogResult => !!result),
      take(1),
    ).subscribe(result => this.onUpdateAnnotation(clickedIndex, result));
  }

  onSave(event: MatChipInputEvent): void {
    this.save(event.value);
    this.onChange(this.annotations);
    this.reset();

    this.control.setValue('');
  }

  onUseSuggestion(key: string): void {
    if (this.disabled || !this.chipInputControl) {
      return;
    }

    const value = `${key}:`;
    this.control.setValue(value);
    this.chipInputControl.nativeElement.focus();
    this.chipInputControl.nativeElement.setSelectionRange(value.length, value.length);
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

  onUpdateAnnotation(index: number, result: AnnotationEditDialogResult): void {
    this.annotations.splice(index, 1, new Annotation({
      key: result.key.trim(),
      value: result.value.trim(),
    }));

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

  protected reset() {
    this.regroup();
    this.cdr.detectChanges();
  }

  protected findAnnotationIndex(key: string, value: string): number {
    return this.annotations.findIndex((element) => {
      return element.key === key && element.value === value;
    });
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
