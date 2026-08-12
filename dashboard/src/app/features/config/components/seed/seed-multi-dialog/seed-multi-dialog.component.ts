import { ChangeDetectionStrategy,Component,inject,OnInit,ViewChild } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { SeedDetailsComponent } from '..';
import { ConfigObject,ConfigRef,Kind,Label } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { LabelMultiComponent } from '../../label/label-multi/label-multi.component';
import {BooleanOverrideComponent} from '../../../../../shared/components';
import {MultiUpdateOperationComponent} from '../../multi-update-operation/multi-update-operation.component';

@Component({
  selector: 'app-seed-multi-dialog',
  templateUrl: './seed-multi-dialog.component.html',
  styleUrls: ['./seed-multi-dialog.component.css', '../../mass-update-dialog.scss'],
  imports: [
    LabelMultiComponent,
    BooleanOverrideComponent,
    MultiUpdateOperationComponent,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class SeedMultiDialogComponent extends SeedDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<SeedMultiDialogComponent>>(MatDialogRef);


  shouldAddLabel = undefined;
  shouldAddCrawlJob = undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {

    super();

    this.configObject = this.data.configObject;
    this.crawlJobs = this.data.options.crawlJobs;
    this.allSelected = this.data.allSelected;
  }


  get labelList() {
    return this.form.get('labelList');
  }

  get updateJobRefListId() {
    return this.form.get('updateJobRefListId');
  }

  get commonJobRefListId() {
    return this.form.get('commonJobRefListId');
  }

  get disabledSelection(): boolean | null {
    return this.form.get('disabledSelection')?.value ?? null;
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || (this.shouldAddLabel !== undefined && this.labelList.value.length)
      || (this.shouldAddCrawlJob !== undefined && this.updateJobRefListId.value.length > 0)
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty
      || this.shouldAddLabel !== undefined
      || this.shouldAddCrawlJob !== undefined;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  override onRevert() {
    this.shouldAddCrawlJob = this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  onToggleShouldAddCrawlJob(shouldAdd: boolean): void {
    if (shouldAdd !== true && shouldAdd !== false) {
      this.shouldAddCrawlJob = undefined;
      this.updateJobRefListId.disable();
      return;
    }
    this.shouldAddCrawlJob = shouldAdd;
    this.updateJobRefListId.patchValue([]);
    this.updateJobRefListId.enable();
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: [[]],
      commonJobRefListId: [[]],
      updateJobRefListId: [{value: [], disabled: true}],
      disabledSelection: null,
    });
  }

  protected override updateForm() {
    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      commonJobRefListId: this.configObject.seed.jobRefList.map(job => job.id),
      updateJobRefListId: [],
      disabledSelection: null,
    });

    this.form.markAsPristine();
    this.form.markAsUntouched();
    this.commonJobRefListId.disable();
    this.updateJobRefListId.disable();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  /**
   * NB: Disabled values in form must be copied from model and not the view model (form.value)
   */
  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
    const formModel = this.form.value;
    const pathList: string[] = [];
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    const seed = updateTemplate.seed;

    if (this.disabledSelection !== null) {
      seed.disabled = formModel.disabledSelection;
      pathList.push('seed.disabled');
    }

    if (this.shouldAddCrawlJob !== undefined) {
      seed.jobRefList = formModel.updateJobRefListId.map(id => new ConfigRef({id, kind: Kind.CRAWLJOB}));
      if (this.shouldAddCrawlJob) {
        pathList.push('seed.jobRef+');
      } else {
        pathList.push('seed.jobRef-');
      }
    }

    if (this.labelList.value.length && this.shouldAddLabel !== undefined) {
      updateTemplate.meta.labelList = formModel.labelList;
      if (this.shouldAddLabel) {
        pathList.push('meta.label+');
      } else {
        pathList.push('meta.label-');
      }
    }
    return {updateTemplate, pathList};
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareMultiSave();
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }
}
