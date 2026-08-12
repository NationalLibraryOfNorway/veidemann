import { ChangeDetectionStrategy,Component,inject,OnInit,ViewChild } from '@angular/core';
import { ReactiveFormsModule,Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { CrawlJobDetailsComponent } from '..';
import { ConfigObject,Kind,Label } from '../../../../../shared/models/config';
import { NUMBER_OR_EMPTY_STRING } from '../../../../../shared/validation/patterns';
import { ConfigDialogData } from '../../../func';
import { DurationPickerComponent } from '../../durationpicker/duration-picker';
import { FilesizeInputComponent } from '../../filesize-input/filesize-input.component';
import { LabelMultiComponent } from '../../label/label-multi/label-multi.component';
import {BooleanOverrideComponent} from '../../../../../shared/components';

@Component({
  selector: 'app-crawljobs-multi-dialog',
  templateUrl: './crawljobs-multi-dialog.component.html',
  styleUrls: ['./crawljobs-multi-dialog.component.css', '../../mass-update-dialog.scss'],
  imports: [
    DurationPickerComponent,
    FilesizeInputComponent,
    LabelMultiComponent,
    BooleanOverrideComponent,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    ReactiveFormsModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlJobMultiDialogComponent extends CrawlJobDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<CrawlJobMultiDialogComponent>>(MatDialogRef);


  shouldAddLabel: boolean = undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {

    super();

    this.configObject = this.data.configObject;
    this.crawlScheduleConfigs = this.data.options.crawlScheduleConfigs;
    this.crawlConfigs = this.data.options.crawlConfigs;
    this.scopeScripts = this.data.options.scopeScripts;
    this.allSelected = this.data.allSelected;
  }

  get labelList() {
    return this.form.get('labelList');
  }

  get disabledSelection(): boolean | null {
    return this.form.get('disabledSelection')?.value ?? null;
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || (this.shouldAddLabel !== undefined && this.labelList.value.length)
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty
      || this.shouldAddLabel !== undefined;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareMultiSave();
  }

  override onRevert() {
    this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: {value: [], disabled: false},
      scheduleRef: this.fb.group({
        id: '',
        kind: Kind.CRAWLSCHEDULECONFIG,
      }),
      crawlConfigRef: this.fb.group({
        id: '',
        kind: Kind.CRAWLCONFIG,
      }),
      scopeScriptRef: this.fb.group({
        id: '',
        kind: Kind.BROWSERSCRIPT,
      }),
      limits: this.fb.group({
        maxDurationS: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
        maxBytes: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      }),
      disabledSelection: null,
    });
  }

  protected override updateForm() {
    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      scheduleRef: this.configObject.crawlJob.scheduleRef,
      crawlConfigRef: this.configObject.crawlJob.crawlConfigRef,
      scopeScriptRef: this.configObject.crawlJob.scopeScriptRef,
      limits: {
        maxDurationS: this.configObject.crawlJob.limits.maxDurationS || '',
        maxBytes: this.configObject.crawlJob.limits.maxBytes || '',
      },
      disabledSelection: null,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
    const pathList: string[] = [];

    const formModel = this.form.value;

    const updateTemplate = new ConfigObject({kind: Kind.CRAWLJOB});
    const crawlJob = updateTemplate.crawlJob;

    if (this.labelList.value.length && this.shouldAddLabel !== undefined) {
      updateTemplate.meta.labelList = formModel.labelList;
      if (this.shouldAddLabel) {
        pathList.push('meta.label+');
      } else {
        pathList.push('meta.label-');
      }
    }

    if (this.disabledSelection !== null) {
      crawlJob.disabled = formModel.disabledSelection;
      pathList.push('crawlJob.disabled');
    }

    if (this.maxBytes.dirty &&
      (this.allSelected || formModel.limits.maxBytes !== this.configObject.crawlJob.limits.maxBytes)) {
      crawlJob.limits.maxBytes = formModel.limits.maxBytes;
      pathList.push('crawlJob.limits.maxBytes');
    }

    if (this.maxDurationSeconds.dirty &&
      (this.allSelected || formModel.limits.maxDurationS !== this.configObject.crawlJob.limits.maxDurationS)) {
      crawlJob.limits.maxDurationS = formModel.limits.maxDurationS;
      pathList.push('crawlJob.limits.maxDurationS');
    }

    if (formModel.scheduleRef.id &&
      this.scheduleRef.dirty && (this.allSelected || formModel.scheduleRef !== this.configObject.crawlJob.scheduleRef)) {
      crawlJob.scheduleRef = formModel.scheduleRef;
      pathList.push('crawlJob.scheduleRef');
    } else {
      updateTemplate.crawlJob.scheduleRef = null;
    }

    if (formModel.crawlConfigRef.id && this.crawlConfigRef.dirty &&
      (this.allSelected || formModel.crawlConfigRef !== this.configObject.crawlJob.crawlConfigRef)) {
      crawlJob.crawlConfigRef = formModel.crawlConfigRef;
      pathList.push('crawlJob.crawlConfigRef');
    } else {
      crawlJob.crawlConfigRef = null;
    }

    if (formModel.scopeScriptRef.id && this.scopeScriptRef.dirty &&
      (this.allSelected || formModel.scopeScriptRef !== this.configObject.crawlJob.scopeScriptRef)) {
      crawlJob.scopeScriptRef = formModel.scopeScriptRef;
      pathList.push('crawlJob.scopeScriptRef');
    } else {
      crawlJob.scopeScriptRef = null;
    }

    return {updateTemplate, pathList};
  }

}
