import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, Input, OnInit, ViewChild, inject } from '@angular/core';
import {
  UntypedFormBuilder,
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  UntypedFormControl,
  UntypedFormGroup,
  Validators,
} from '@angular/forms';
import {BehaviorSubject, combineLatest, Observable, Subject} from 'rxjs';
import {COMMA, ENTER} from '@angular/cdk/keycodes';
import {Kind, Label} from '../../../../shared/models';
import {filter, map, startWith, switchMap, take} from 'rxjs/operators';
import {MatChipInputEvent, MatChipsModule} from '@angular/material/chips';
import {LabelService} from '../../services/label.service';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatAutocompleteModule} from '@angular/material/autocomplete';
import {CdkDrag, CdkDropList} from '@angular/cdk/drag-drop';
import {AsyncPipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {MatInput} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';
import {MatDialog} from '@angular/material/dialog';
import {MatTooltipModule} from '@angular/material/tooltip';
import {EMOJI_LABEL_KEY, LabelDisplayComponent} from '../../../../shared/components';
import {
  AnnotationEditDialogComponent,
  AnnotationEditDialogResult,
} from '../annotation/annotation-edit-dialog/annotation-edit-dialog.component';

interface LabelGroup {
  key: string;
  values: string[];
}


@Component({
  selector: 'app-labels',
  templateUrl: './label.component.html',
  styleUrls: ['./label.component.scss'],
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: LabelComponent, multi: true}],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    CdkDrag,
    CdkDropList,
    MatAutocompleteModule,
    MatButtonModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatTooltipModule,
    LabelDisplayComponent,
    ReactiveFormsModule
  ],
  standalone: true
})

export class LabelComponent implements ControlValueAccessor, OnInit {
  protected fb = inject(UntypedFormBuilder);
  protected cdr = inject(ChangeDetectorRef);
  protected labelService = inject(LabelService);
  private dialog = inject(MatDialog);


  @Input()
  removable = true;

  @Input()
  placeholderText = 'New label...';

  @Input({required: true})
  kind: Kind;
  labelText = 'Label';

  protected emojiPickerEnabled = true;

  private fetchLabelKeys: Subject<void>;

  control = new UntypedFormControl();
  protected labelForm: UntypedFormGroup;

  // ControlValueAccessor callback functions
  onChange: (labels: Label[]) => void;
  onTouched: () => void;

  labelInputSeparators = [ENTER, COMMA];

  disabled = false;

  // hack to sequence events between matAutocomplete and matChipList
  // see: https://github.com/angular/components/issues/8176
  protected seq = false;


  protected groupsSubject = new BehaviorSubject<LabelGroup[]>([]);
  groups$: Observable<LabelGroup[]> = this.groupsSubject.asObservable();

  protected labels: Label[];

  filteredKey$: Observable<string[]>;

  @ViewChild('chipInput', {static: true}) chipInputControl: ElementRef;

  constructor() {
    this.createForm();
    this.fetchLabelKeys = new Subject();
  }

  ngOnInit(): void {
    const value$ = this.control.valueChanges.pipe(
      startWith(''),
      map(value => value || '')
    );
    const key$ = this.fetchLabelKeys.pipe(
      startWith(''),
      switchMap(() => this.labelService.getLabelKeys(this.kind))
    );
    this.filteredKey$ = combineLatest([value$, key$])
      .pipe(
        map(([value, keys]) => {
          const filterValue = value.toLowerCase();
          return keys.filter(key => key.toLowerCase().startsWith(filterValue));
        })
      );
  }

  onAutocompleteOptionSelected(event) {
    this.seq = true;
    this.chipInputControl.nativeElement.value = event.option.value;
  }

  // implement ControlValueAccessor
  writeValue(labels: Label[]): void {
    this.fetchLabelKeys.next();
    if (labels === null) {
      this.labels = [];
    } else {
      this.labels = labels.map(label => new Label({key: label.key, value: label.value}));
    }
    this.reset();
  }

  // implement ControlValueAccessor
  registerOnChange(fn: (labels: Label[]) => void): void {
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

  onClickLabel(key: string, value: string): void {
    if (this.disabled) {
      return;
    }
    const index = this.findLabelIndex(key, value);
    if (index < 0) return;
    this.dialog.open<AnnotationEditDialogComponent, {key: string; value: string; type: 'label'}, AnnotationEditDialogResult>(
      AnnotationEditDialogComponent,
      {data: {key, value, type: 'label'}, width: '480px', maxWidth: 'calc(100vw - 32px)', autoFocus: false}
    ).afterClosed().pipe(
      filter((result): result is AnnotationEditDialogResult => !!result),
      take(1),
    ).subscribe(result => this.onUpdateLabel(index, result.key, result.value));
  }

  onDrop(event) {
    const label = event.item.data;
    this.save(label);
  }

  onNativeDragOver(event: DragEvent): void {
    if (this.disabled || !event.dataTransfer?.types.includes('text/plain')) {
      return;
    }
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
  }

  onNativeDrop(event: DragEvent): void {
    if (this.disabled) {
      return;
    }
    event.preventDefault();
    const previousLength = this.labels.length;
    this.save(event.dataTransfer?.getData('text/plain') ?? '');
    if (this.labels.length === previousLength) {
      return;
    }
    this.onChange(this.labels);
    this.reset();
  }

  async onChooseEmoji(): Promise<void> {
    if (this.disabled || !this.emojiPickerEnabled) {
      return;
    }

    const {EmojiPickerDialogComponent} = await import('../../../../shared/components/emoji-picker/emoji-picker-dialog.component');
    this.dialog.open<InstanceType<typeof EmojiPickerDialogComponent>, void, string>(EmojiPickerDialogComponent, {
      width: '552px',
      maxWidth: 'calc(100vw - 24px)',
      autoFocus: false,
      restoreFocus: true,
    }).afterClosed().pipe(
      filter((unicode): unicode is string => !!unicode),
      take(1),
    ).subscribe(unicode => this.onEmojiSelected(unicode));
  }

  onSave(event: MatChipInputEvent): void {
    if (this.seq) {
      this.seq = false;
      return;
    }

    this.save(event.value);

    this.onChange(this.labels);
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

    if (this.findLabelIndex(key, value) > -1) {
      return;
    }

    this.labels.push(new Label({key, value}));
  }

  private onEmojiSelected(unicode: string): void {
    const previousLength = this.labels.length;
    this.save(`${EMOJI_LABEL_KEY}:${unicode}`);
    if (this.labels.length === previousLength) {
      return;
    }
    this.onChange(this.labels);
    this.reset();
    this.control.setValue('');
    this.chipInputControl.nativeElement.value = '';
  }

  onUpdateLabel(index: number, key: string, value: string): void {
    key = key.trim();
    value = value.trim();

    // remove old
    this.labels.splice(index, 1);
    // add updated
    this.labels.push(new Label({key, value}));

    this.onChange(this.labels);
    this.reset();
  }

  onRemoveLabel(key: string, value: string): void {
    const index = this.findLabelIndex(key, value);
    if (index !== -1) {
      this.labels.splice(index, 1);
    }
    this.onChange(this.labels);
    this.reset();
  }

  removeLabelAriaLabel(key: string, value: string): string {
    return $localize`:@@removeLabelButtonLabel:Remove label ${key}:LABEL_KEY: ${value}:LABEL_VALUE:`;
  }

  protected reset() {
    this.regroup();
    this.labelForm.reset();
    this.labelForm.disable();
    this.cdr.detectChanges();
  }

  protected findLabelIndex(key: string, value: string): number {
    return this.labels.findIndex((element) => {
      return element.key === key && element.value === value;
    });
  }

  protected createForm(): void {
    this.labelForm = this.fb.group({key: ['', Validators.required], value: ['', Validators.required]});
    this.labelForm.disable();
  }

  // group labels with similar key together
  protected regroup(): void {
    const grouping: Record<string, string[]> = {};

    this.labels.forEach(label => {
      if (Object.hasOwn(grouping, label.key)) {
        grouping[label.key].push(label.value);
      } else {
        grouping[label.key] = [label.value];
      }
    });
    this.groupsSubject.next(Object.keys(grouping).map(key => ({key, values: grouping[key]})));
  }
}
