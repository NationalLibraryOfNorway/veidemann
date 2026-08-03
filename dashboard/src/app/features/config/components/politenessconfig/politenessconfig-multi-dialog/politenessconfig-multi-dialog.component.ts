import { ChangeDetectionStrategy,Component,inject,OnInit,ViewChild } from '@angular/core';
import { AbstractControl,ReactiveFormsModule,Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckbox } from '@angular/material/checkbox';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTooltip } from '@angular/material/tooltip';
import { FlexDirective,LayoutDirective } from '@ngbracket/ngx-layout';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { PolitenessConfigDetailsComponent } from '..';
import { ConfigObject,Kind,Label,RobotsPolicy } from '../../../../../shared/models/config';
import { NUMBER_OR_EMPTY_STRING } from '../../../../../shared/validation/patterns';
import { ConfigDialogData } from '../../../func';
import { DurationPickerComponent } from '../../durationpicker/duration-picker';
import { LabelMultiComponent } from '../../label/label-multi/label-multi.component';

@Component({
  selector: 'app-politenessconfig-multi-dialog',
  templateUrl: './politenessconfig-multi-dialog.component.html',
  styleUrls: ['./politenessconfig-multi-dialog.component.css'],
  imports: [
    DurationPickerComponent,
    FlexDirective,
    LabelMultiComponent,
    LayoutDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatCheckbox,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTooltip,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class PolitenessConfigMultiDialogComponent extends PolitenessConfigDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<PolitenessConfigMultiDialogComponent>>(MatDialogRef);


  allSelected = false;
  shouldAddLabel = undefined;
  shouldAddSelector = undefined;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {

    super();

    this.configObject = this.data.configObject;
    this.robotsPolicies = this.data.options.robotsPolicies;
    this.allSelected = this.data.allSelected;
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || (this.shouldAddLabel !== undefined && this.labelList.value.length)
      || (this.robotsPolicy.enabled && (this.allSelected || this.configObject.politenessConfig.robotsPolicy === undefined))
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty
      || (this.robotsPolicy.enabled && this.configObject.politenessConfig.robotsPolicy === undefined)
      || (this.shouldAddLabel !== undefined);
  }

  get labelList(): AbstractControl {
    return this.form.get('labelList');
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onEnableRobotsPolicy() {
    if (this.robotsPolicy.disabled) {
      this.robotsPolicy.enable();
    }
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }

  override onRevert() {
    this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: {value: [], disabled: false},
      robotsPolicy: '',
      minimumRobotsValidityDurationS: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      customRobots: null,
      useHostname: {value: '', disabled: true},
    });
  }

  protected override updateForm() {
    if (this.configObject.politenessConfig.useHostname !== null && !this.allSelected) {
      this.useHostname.enable();
    } else {
      this.useHostname.disable();
    }

    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      robotsPolicy: this.configObject.politenessConfig.robotsPolicy || RobotsPolicy.OBEY_ROBOTS,
      minimumRobotsValidityDurationS: this.configObject.politenessConfig.minimumRobotsValidityDurationS || '',
      customRobots: this.configObject.politenessConfig.customRobots,
      useHostname: this.configObject.politenessConfig.useHostname,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (this.configObject.politenessConfig.robotsPolicy === undefined) {
      this.robotsPolicy.disable();
    }
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
    const pathList: string[] = [];

    const formModel = this.form.value;

    const updateTemplate = new ConfigObject({kind: Kind.POLITENESSCONFIG});
    const politenessConfig = updateTemplate.politenessConfig;

    if (this.labelList.value.length && this.shouldAddLabel !== undefined) {
      updateTemplate.meta.labelList = formModel.labelList;
      if (this.shouldAddLabel) {
        pathList.push('meta.label+');
      } else {
        pathList.push('meta.label-');
      }
    }

    if (this.robotsPolicy.enabled && (this.allSelected || formModel.robotsPolicy !== this.configObject.politenessConfig.robotsPolicy)) {
      politenessConfig.robotsPolicy = formModel.robotsPolicy;
      pathList.push('politenessConfig.robotsPolicy');
    }

    if (this.customRobots.dirty && (this.allSelected || formModel.customRobots !== this.configObject.politenessConfig.customRobots)) {
      politenessConfig.customRobots = formModel.customRobots;
      pathList.push('politenessConfig.customRobots');

    }

    if (this.minRobotsValidDurationSec.dirty
      && (this.allSelected
        || formModel.minimumRobotsValidityDurationS !== this.configObject.politenessConfig.minimumRobotsValidityDurationS)) {
      politenessConfig.minimumRobotsValidityDurationS = formModel.minimumRobotsValidityDurationS;
      pathList.push('politenessConfig.minimumRobotsValidityDurationS');
    }

    if (this.useHostname.dirty && (this.allSelected || formModel.useHostname !== this.configObject.politenessConfig.useHostname)) {
      politenessConfig.useHostname = formModel.useHostname;
      pathList.push('politenessConfig.useHostname');
    }

    return {updateTemplate, pathList};
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareMultiSave();
  }
}
