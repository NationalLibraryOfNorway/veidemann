import {ChangeDetectionStrategy, Component, Inject, OnInit, ViewChild} from '@angular/core';
import {BrowserConfigDetailsComponent} from '..';
import {AbstractControl, ReactiveFormsModule, UntypedFormBuilder, Validators} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {ConfigObject, ConfigRef, Kind, Label} from '../../../../../shared/models';
import {AuthService} from '../../../../../core/auth';
import {NUMBER_OR_EMPTY_STRING} from '../../../../../shared/validation/patterns';
import {MatFormFieldModule} from '@angular/material/form-field';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MatSelectModule} from '@angular/material/select';
import {MatButtonToggleModule} from '@angular/material/button-toggle';
import {SelectorComponent} from '../../selector/selector.component';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-browserconfig-multi-dialog',
  templateUrl: './browserconfig-multi-dialog.component.html',
  styleUrls: ['./browserconfig-multi-dialog.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    LabelMultiComponent,
    ReactiveFormsModule,
    MatFormFieldModule,
    DurationPickerComponent,
    MatDialogModule,
    MatButtonModule,
    MatChipsModule,
    MatSelectModule,
    MatButtonToggleModule,
    SelectorComponent,
    MatIconModule,
    MatInputModule,
    LayoutDirective,
    FlexDirective,
    LayoutGapDirective
  ],
  standalone: true
})

export class BrowserConfigMultiDialogComponent extends BrowserConfigDetailsComponent implements OnInit {
  override readonly Kind = Kind;

  allSelected = false;

  shouldAddLabel = undefined;
  shouldAddBrowserScript = undefined;
  shouldAddSelector = undefined;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<BrowserConfigMultiDialogComponent>) {
    super(fb, authService);
    this.configObject = this.data.configObject;
    this.browserScripts = this.data.options.browserScripts;
    this.allSelected = this.data.allSelected;
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty
      || ((this.shouldAddBrowserScript !== undefined && this.scriptRefIdList.value.length)
        || (this.shouldAddSelector !== undefined && this.scriptSelectorList.value.length)
        || (this.shouldAddLabel !== undefined && this.labelList.value.length)
      ));
  }

  override get canRevert(): boolean {
    return this.form.dirty
      || (this.shouldAddLabel !== undefined)
      || (this.shouldAddBrowserScript !== undefined)
      || (this.shouldAddSelector !== undefined);
  }

  get labelList(): AbstractControl {
    return this.form.get('labelList');
  }

  get commonScriptSelectorList(): AbstractControl {
    return this.form.get('commonScriptSelectorList');
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onToggleShouldAddSelector(value: boolean) {
    this.shouldAddSelector = value;
    this.scriptSelectorList.patchValue([]);
  }

  onToggleBrowserScript(value: boolean) {
    this.shouldAddBrowserScript = value;
    this.scriptRefIdList.patchValue([]);
  }

  override onRevert() {
    super.onRevert();
    this.shouldAddLabel = undefined;
    this.shouldAddBrowserScript = undefined;
    this.shouldAddSelector = undefined;
    this.labelMulti.onRevert();
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
      userAgent: '',
      windowWidth: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      windowHeight: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      pageLoadTimeoutMs: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      maxInactivityTimeMs: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      // headers: this.fb.group({''}),
      commonScriptRefIdList: [[]],
      scriptRefIdList: [[]],
      scriptSelectorList: '',
    });
  }

  protected override updateForm() {
    this.form.setValue({
      labelList: this.configObject.meta.labelList,
      userAgent: this.configObject.browserConfig.userAgent || '',
      windowWidth: this.configObject.browserConfig.windowWidth || '',
      windowHeight: this.configObject.browserConfig.windowHeight || '',
      pageLoadTimeoutMs: this.configObject.browserConfig.pageLoadTimeoutMs || '',
      maxInactivityTimeMs: this.configObject.browserConfig.maxInactivityTimeMs || '',
      // headers: this.configObject.configObject.headers;
      commonScriptRefIdList: this.configObject.browserConfig.scriptRefList.map(ref => ref.id),
      scriptRefIdList: [],
      scriptSelectorList: [[]],
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }


  protected override prepareSave(): any {
    const pathList: string[] = [];
    const formModel = this.form.value;

    const updateTemplate = new ConfigObject({kind: Kind.BROWSERCONFIG});
    const browserConfig = updateTemplate.browserConfig;


    if (this.userAgent.dirty && (this.allSelected || formModel.userAgent !== this.configObject.browserConfig.userAgent)) {
      browserConfig.userAgent = formModel.userAgent;
      pathList.push('browserConfig.userAgent');
    }

    if (this.windowWidth.dirty && (this.allSelected || this.configObject.browserConfig.windowWidth !== formModel.windowWidth)) {
      browserConfig.windowWidth = formModel.windowWidth;
      pathList.push('browserConfig.windowWidth');
    }

    if (this.windowHeight.dirty && (this.allSelected || this.configObject.browserConfig.windowHeight !== formModel.windowHeight)) {
      browserConfig.windowHeight = formModel.windowHeight;
      pathList.push('browserConfig.windowHeight');
    }

    if (this.pageLoadTimeoutMs.dirty
      && (this.allSelected || this.configObject.browserConfig.pageLoadTimeoutMs !== formModel.pageLoadTimeoutMs)) {
      browserConfig.pageLoadTimeoutMs = formModel.pageLoadTimeoutMs;
      pathList.push('browserConfig.pageLoadTimeoutMs');
    }

    if (this.maxInactivityTimeMs.dirty
      && (this.allSelected || this.configObject.browserConfig.maxInactivityTimeMs !== formModel.maxInactivityTimeMs)) {
      browserConfig.maxInactivityTimeMs = formModel.maxInactivityTimeMs;
      pathList.push('browserConfig.maxInactivityTimeMs');
    }

    if (this.labelList.value.length && this.shouldAddLabel !== undefined) {
      updateTemplate.meta.labelList = formModel.labelList;
      if (this.shouldAddLabel) {
        pathList.push('meta.label+');
      } else {
        pathList.push('meta.label-');
      }
    }

    if (this.shouldAddBrowserScript !== undefined) {
      browserConfig.scriptRefList = formModel.scriptRefIdList.map(id => new ConfigRef({id, kind: Kind.BROWSERSCRIPT}));
      if (this.shouldAddBrowserScript) {
        pathList.push('browserConfig.scriptRef+');
      } else {
        pathList.push('browserConfig.scriptRef-');
      }
    }

    if (this.scriptSelectorList.value.length && this.shouldAddSelector !== undefined) {
      browserConfig.scriptSelectorList = formModel.scriptSelectorList.map(label => label.key + ':' + label.value);
      if (this.shouldAddSelector) {
        pathList.push('browserConfig.scriptSelector+');
      } else {
        pathList.push('browserConfig.scriptSelector-');
      }
    }

    return {updateTemplate, pathList};
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareSave();
  }

}
