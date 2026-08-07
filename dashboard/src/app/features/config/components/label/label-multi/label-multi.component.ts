import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  EventEmitter,
  inject,
  Input,
  OnInit,
  Output,
  QueryList,
  ViewChild,
  ViewChildren
} from '@angular/core';
import {ConfigObject, Label} from '../../../../../shared/models/config';
import {ENTER} from '@angular/cdk/keycodes';
import {MatChipInputEvent, MatChipOption, MatChipSelectionChange, MatChipsModule} from '@angular/material/chips';
import {combineLatest, Observable, Subject} from 'rxjs';
import {CdkDrag, CdkDragDrop, CdkDropList} from '@angular/cdk/drag-drop';
import {filter, map, startWith, switchMap, take} from 'rxjs/operators';
import {LabelService} from '../../../services/label.service';
import {ReactiveFormsModule, UntypedFormControl} from '@angular/forms';
import {MatButtonToggleModule} from '@angular/material/button-toggle';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatAutocompleteModule} from '@angular/material/autocomplete';
import {AsyncPipe} from '@angular/common';

export interface LabelUpdate {
  add: boolean;
  labels: Label[];
}
import {MatTooltipModule} from '@angular/material/tooltip';
import {MatDialog} from '@angular/material/dialog';
import {EMOJI_LABEL_KEY, LabelDisplayComponent} from '../../../../../shared/components';

@Component({
  selector: 'app-label-multi',
  templateUrl: './label-multi.component.html',
  styleUrls: ['./label-multi.component.css'],
  imports: [
    AsyncPipe,
    CdkDrag,
    CdkDropList,
    MatAutocompleteModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    LabelDisplayComponent,
    MatTooltipModule,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class LabelMultiComponent implements OnInit {
  protected labelService = inject(LabelService);
  private dialog = inject(MatDialog);

  @Input()
  configObject: ConfigObject;

  @Input()
  allSelected: boolean;

  @Output()
  update = new EventEmitter<LabelUpdate>();

  private fetchLabelKeys: Subject<void>;

  control = new UntypedFormControl();

  shouldAddLabel = undefined;
  labelInputSeparators = [ENTER];
  labels: Label[] = [];
  filteredKey$: Observable<string[]>;

  // hack to sequence events between matAutocomplete and matChipList
  // see: https://github.com/angular/components/issues/8176
  protected seq = false;

  @ViewChild('chipInput') chipInputControl: ElementRef;
  @ViewChildren(MatChipOption) commonLabelOptions: QueryList<MatChipOption>;

  constructor() {
    this.fetchLabelKeys = new Subject();
  }

  ngOnInit(): void {
    const value$ = this.control.valueChanges.pipe(
      startWith(''),
      map(value => value || '')
    );
    const key$ = this.fetchLabelKeys.pipe(
      startWith(''),
      switchMap(() => this.labelService.getLabelKeys())
    );
    this.filteredKey$ = combineLatest([value$, key$])
      .pipe(
        map(([value, keys]) => {
          const filterValue = value.toLowerCase();
          return keys.filter(key => key.toLowerCase().startsWith(filterValue));
        })
      );
  }

  onToggleShouldAddLabels(shouldAdd: boolean): void {
    if (this.shouldAddLabel !== undefined) {
      this.labels = [];
      this.clearCommonLabelSelection();
    }
    this.shouldAddLabel = shouldAdd;
    this.update.emit({add: this.shouldAddLabel, labels: this.labels});
  }

  onAdd(event: MatChipInputEvent) {
    if (event.chipInput) {
      event.input.value = '';
    }

    let key = '';
    let value = event.value.trim();

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

    this.addLabel(key, value);
  }

  async onChooseEmoji(): Promise<void> {
    if (this.shouldAddLabel !== true) {
      return;
    }

    const {EmojiPickerDialogComponent} = await import('../../../../../shared/components/emoji-picker/emoji-picker-dialog.component');
    this.dialog.open<InstanceType<typeof EmojiPickerDialogComponent>, void, string>(EmojiPickerDialogComponent, {
      width: '464px',
      maxWidth: 'calc(100vw - 24px)',
      autoFocus: false,
      restoreFocus: true,
    }).afterClosed().pipe(
      filter((unicode): unicode is string => !!unicode),
      take(1),
    ).subscribe(unicode => this.addLabel(EMOJI_LABEL_KEY, unicode));
  }

  onRemove(key: string, value: string) {
    const index = this.findLabelIndex(key, value);
    if (index !== -1) {
      this.labels.splice(index, 1);
      this.findCommonLabelOption(key, value)?.deselect();
      this.update.emit({add: this.shouldAddLabel, labels: this.labels});
    }
  }

  onCommonLabelSelectionChange(label: Label, event: MatChipSelectionChange): void {
    if (!event.isUserInput || this.shouldAddLabel === undefined) {
      return;
    }

    if (event.selected) {
      this.addLabel(label.key, label.value);
    } else {
      this.onRemove(label.key, label.value);
    }
  }

  protected findLabelIndex(key: string, value: string): number {
    return this.labels.findIndex((element) => {
      return element.key === key && element.value === value;
    });
  }

  private addLabel(key: string, value: string): void {
    if (this.findLabelIndex(key, value) > -1) {
      return;
    }

    this.labels.push(new Label({key, value}));
    this.findCommonLabelOption(key, value)?.select();
    this.update.emit({add: this.shouldAddLabel, labels: this.labels});
  }

  onRevert() {
    this.shouldAddLabel = undefined;
    this.labels = [];
    this.clearCommonLabelSelection();
  }

  onDrop(event: CdkDragDrop<string[]>): void {
    const label: MatChipInputEvent = {input: undefined, chipInput: null, value: event.item.data};
    this.onAdd(label);
  }

  onAutocompleteOptionSelected(event) {
    this.seq = true;
    this.chipInputControl.nativeElement.value = event.option.value;
  }

  private findCommonLabelOption(key: string, value: string): MatChipOption | undefined {
    const optionValue = `${key}:${value}`;
    return this.commonLabelOptions?.find(option => option.value === optionValue);
  }

  private clearCommonLabelSelection(): void {
    this.commonLabelOptions?.forEach(option => option.deselect());
  }

}
