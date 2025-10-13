import {Component, Inject, OnInit, ViewChild} from '@angular/core';
import {SeedDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, ConfigRef, Kind, Label} from '../../../../../shared/models/config';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {MatSlideToggle} from '@angular/material/slide-toggle';
import {MatTooltip} from '@angular/material/tooltip';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {MatButtonModule} from '@angular/material/button';
import {MatButtonToggleModule} from '@angular/material/button-toggle';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatIcon} from '@angular/material/icon';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';

@Component({
  selector: 'app-seed-multi-dialog',
  templateUrl: './seed-multi-dialog.component.html',
  styleUrls: ['./seed-multi-dialog.component.css'],
  imports: [
    FlexDirective,
    LabelMultiComponent,
    LayoutDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatButtonToggleModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIcon,
    MatSelectModule,
    MatSlideToggle,
    MatTooltip,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class SeedMultiDialogComponent extends SeedDetailsComponent implements OnInit {

  shouldAddLabel = undefined;
  shouldAddCrawlJob = undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<SeedMultiDialogComponent>) {
    super(fb, authService);
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

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || (this.shouldAddLabel !== undefined && this.labelList.value.length)
      || (this.shouldAddCrawlJob !== undefined && this.updateJobRefListId.value.length > 0)
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty || this.shouldAddLabel !== undefined || this.shouldAddCrawlJob !== undefined;
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
    this.shouldAddCrawlJob = shouldAdd;
    this.updateJobRefListId.patchValue([]);
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: [[]],
      disabled: {value: '', disabled: true},
      commonJobRefListId: [[]],
      updateJobRefListId: [[]]
    });
  }

  protected override updateForm() {
    if (this.configObject.seed.disabled !== undefined) {
      this.disabled.enable();
    } else {
      this.disabled.disable();
    }

    if (this.allSelected) {
      this.disabled.disable();
    }

    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      disabled: !!this.configObject.seed.disabled,
      commonJobRefListId: this.configObject.seed.jobRefList.map(job => job.id),
      updateJobRefListId: []
    });

    this.form.markAsPristine();
    this.form.markAsUntouched();
    this.commonJobRefListId.disable();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  /**
   * NB: Disabled values in form must be copied from model and not the view model (form.value)
   */
  protected override prepareSave(): any {
    const formModel = this.form.value;
    const pathList: string[] = [];
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    const seed = updateTemplate.seed;

    if (this.disabled.dirty && formModel.disabled !== undefined) {
      seed.disabled = formModel.disabled;
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
    return this.prepareSave();
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }
}
