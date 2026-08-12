import {ChangeDetectionStrategy, Component, inject, OnInit, ViewChild} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {CollectionDetailsComponent} from '..';
import {BooleanOverrideComponent} from '../../../../../shared/components';
import {ConfigObject, Kind, Label, RotationPolicy} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {FilesizeInputComponent} from '../../filesize-input/filesize-input.component';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';

@Component({
  selector: 'app-collection-multi-dialog',
  templateUrl: './collection-multi-dialog.component.html',
  styleUrls: ['../../mass-update-dialog.scss'],
  imports: [
    BooleanOverrideComponent,
    FilesizeInputComponent,
    LabelMultiComponent,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class CollectionMultiDialogComponent extends CollectionDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<CollectionMultiDialogComponent>>(MatDialogRef);

  shouldAddLabel: boolean | undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {
    super();
    this.configObject = this.data.configObject;
    this.rotationPolicies = this.data.options.rotationPolicies ?? [];
    this.allSelected = this.data.allSelected ?? false;
  }

  get labelList() {
    return this.form.get('labelList');
  }

  get collectionDedupPolicy() {
    return this.form.get('collectionDedupPolicy');
  }

  get fileRotationPolicy() {
    return this.form.get('fileRotationPolicy');
  }

  get compress() {
    return this.form.get('compress');
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || (this.shouldAddLabel !== undefined && this.labelList.value.length > 0)
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty || this.shouldAddLabel !== undefined;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onUpdateLabels({add, labels}: {add: boolean; labels: Label[]}): void {
    this.shouldAddLabel = add;
    this.labelList.patchValue(labels);
  }

  override onRevert(): void {
    this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      labelList: [[]],
      collectionDedupPolicy: null,
      fileRotationPolicy: null,
      fileSize: '',
      compress: null,
    });
  }

  protected override updateForm(): void {
    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      collectionDedupPolicy: this.configObject.collection.collectionDedupPolicy,
      fileRotationPolicy: this.configObject.collection.fileRotationPolicy,
      fileSize: Number.isNaN(this.configObject.collection.fileSize) ? '' : this.configObject.collection.fileSize,
      compress: null,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) this.form.disable();
  }

  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
    const value = this.form.getRawValue();
    const updateTemplate = new ConfigObject({kind: Kind.COLLECTION});
    const pathList: string[] = [];

    if (this.shouldAddLabel !== undefined && value.labelList.length > 0) {
      updateTemplate.meta.labelList = value.labelList;
      pathList.push(this.shouldAddLabel ? 'meta.label+' : 'meta.label-');
    }
    if (this.collectionDedupPolicy.dirty
      && (this.allSelected || value.collectionDedupPolicy !== this.configObject.collection.collectionDedupPolicy)) {
      updateTemplate.collection.collectionDedupPolicy = value.collectionDedupPolicy;
      pathList.push('collection.collectionDedupPolicy');
    }
    if (this.fileRotationPolicy.dirty
      && (this.allSelected || value.fileRotationPolicy !== this.configObject.collection.fileRotationPolicy)) {
      updateTemplate.collection.fileRotationPolicy = value.fileRotationPolicy;
      pathList.push('collection.fileRotationPolicy');
    }
    if (this.fileSize.dirty && (this.allSelected || value.fileSize !== this.configObject.collection.fileSize)) {
      updateTemplate.collection.fileSize = value.fileSize;
      pathList.push('collection.fileSize');
    }
    if (this.compress.dirty && value.compress !== null) {
      updateTemplate.collection.compress = value.compress;
      pathList.push('collection.compress');
    }

    return {updateTemplate, pathList};
  }

  onDialogClose(): {updateTemplate: ConfigObject; pathList: string[]} {
    return this.prepareMultiSave();
  }

  override readonly RotationPolicy = RotationPolicy;
}
