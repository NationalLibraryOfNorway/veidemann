import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject } from '@angular/core';
import {
  AbstractControl,
  ReactiveFormsModule,
  UntypedFormArray,
  UntypedFormBuilder,
  UntypedFormGroup,
  Validators
} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {CrawlHostGroupConfigIpValidation} from './crawlhostgroupconfig-ipvalidation';
import {ConfigObject, CrawlHostGroupConfig, Kind, Meta} from '../../../../../shared/models';
import {IpRange} from '../../../../../shared/models/config/ip-range.model';
import {ANY_DECIMAL_NUMBER_OR_EMPTY_STRING, NUMBER_OR_EMPTY_STRING} from '../../../../../shared/validation/patterns';
import {UnitOfTime} from '../../../../../shared/models/duration/unit-time.model';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MetaComponent} from '../../meta/meta.component';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';
import {MatTooltip} from '@angular/material/tooltip';
import {MatChipsModule} from '@angular/material/chips';
import {CopyIdDirective} from '../../../../../shared/directives';

@Component({
  selector: 'app-crawlhostgroupconfig-details',
  templateUrl: './crawlhostgroupconfig-details.component.html',
  styleUrls: ['./crawlhostgroupconfig-details.component.scss', '../../config-details-grid.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CopyIdDirective,
    MatButtonModule,
    MatChipsModule,
    MatFormFieldModule,
    MatInputModule,
    MatIcon,
    MatTooltip,
    DurationPickerComponent,
    MetaComponent,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class CrawlHostGroupConfigDetailsComponent implements OnChanges {
  protected fb = inject(UntypedFormBuilder);
  protected authService = inject(AuthService);

  readonly UnitOfTime = UnitOfTime;

  @Input()
  configObject: ConfigObject;

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  update = new EventEmitter<ConfigObject>();

  form: UntypedFormGroup;

  constructor() {
    this.createForm();
  }

  get canEdit(): boolean {
    return this.authService.canUpdate(this.configObject.kind);
  }

  get showSave(): boolean {
    return this.configObject && !this.configObject.id;
  }

  get canSave(): boolean {
    return this.form.valid
      && this.allIpRangesValid();
  }

  get canUpdate(): boolean {
    return this.form.valid
      && this.form.dirty
      && this.allIpRangesValid();
  }

  get canRevert(): boolean {
    return this.form.dirty;
  }

  get name(): string {
    return this.form.get('meta').value.name;
  }

  get ipRangeControlArray(): UntypedFormArray {
    return this.form.get('ipRangeList') as UntypedFormArray;
  }

  get minTimeBetweenPageloadMs(): AbstractControl {
    return this.form.get('minTimeBetweenPageLoadMs');
  }

  get maxTimeBetweenPageloadMs(): AbstractControl {
    return this.form.get('maxTimeBetweenPageLoadMs');
  }

  get delayFactor(): AbstractControl {
    return this.form.get('delayFactor');
  }

  get maxRetries(): AbstractControl {
    return this.form.get('maxRetries');
  }

  get retryDelaySeconds(): AbstractControl {
    return this.form.get('retryDelaySeconds');
  }

  ipFromControl(index: number): AbstractControl {
    return this.form.get(['ipRangeList', index, 'ipFrom']);
  }

  ipToControl(index: number): AbstractControl {
    return this.form.get(['ipRangeList', index, 'ipTo']);

  }

  isValidIpRange(fromIp: string, toIp: string): boolean {
    return CrawlHostGroupConfigIpValidation.isValidRange(fromIp, toIp);
  }

  shouldShowControlError(control: AbstractControl): boolean {
    return control.invalid && (control.dirty || control.touched);
  }

  shouldShowInvalidRange(index: number): boolean {
    const from = this.ipFromControl(index);
    const to = this.ipToControl(index);
    return !!from.value && !!to.value
      && (from.dirty || from.touched || to.dirty || to.touched)
      && !this.isValidIpRange(from.value, to.value);
  }


  ngOnChanges(changes: SimpleChanges) {
    if (changes['configObject']) {
      if (!this.configObject) {
        console.log('changes cfg obj: ', this.configObject);
        this.form.reset();
      } else {
        this.updateForm();
      }
    }
  }

  onSave() {
    this.save.emit(this.prepareSave());
  }

  onUpdate(): void {
    this.update.emit(this.prepareSave());
  }

  onRevert() {
    this.updateForm();
  }

  onAddIpRange() {
    this.ipRangeControlArray.push(this.initIpRange());
  }

  onAddIpRangeKeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    this.onAddIpRange();
  }

  onRemoveIpRange(i: number) {
    this.ipRangeControlArray.removeAt(i);
    this.form.markAsDirty();
  }

  removeIpRangeLabel(index: number): string {
    return $localize`:@@crawlhostgroupconfigRemoveIpRangeAriaLabel:Remove IP range ${index + 1}`;
  }

  protected createForm() {
    this.form = this.fb.group({
      id: '',
      ipRangeList: this.fb.array([]),
      minTimeBetweenPageLoadMs: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      maxTimeBetweenPageLoadMs: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      delayFactor: ['', [Validators.pattern(ANY_DECIMAL_NUMBER_OR_EMPTY_STRING)]],
      maxRetries: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      retryDelaySeconds: ['', [Validators.pattern(NUMBER_OR_EMPTY_STRING)]],
      meta: new Meta(),
    });
  }

  protected updateForm() {
    const ipRangeFG: UntypedFormGroup[] = this.configObject.crawlHostGroupConfig.ipRangeList
      .map(ipRange => this.initIpRange(ipRange));

    const ipRangeFGArray: UntypedFormArray = this.fb.array(ipRangeFG);
    if (this.form.disabled) {
      ipRangeFGArray.disable();
    }
    this.form.patchValue({
      id: this.configObject.id,
      minTimeBetweenPageLoadMs: this.configObject.crawlHostGroupConfig.minTimeBetweenPageLoadMs || '',
      maxTimeBetweenPageLoadMs: this.configObject.crawlHostGroupConfig.maxTimeBetweenPageLoadMs || '',
      delayFactor: this.configObject.crawlHostGroupConfig.delayFactor || '',
      maxRetries: this.configObject.crawlHostGroupConfig.maxRetries || '',
      retryDelaySeconds: this.configObject.crawlHostGroupConfig.retryDelaySeconds || '',
      meta: this.configObject.meta,
    });
    this.form.setControl('ipRangeList', ipRangeFGArray);
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareSave(): ConfigObject {
    const formModel = this.form.value;

    const configObject = new ConfigObject({kind: Kind.CRAWLHOSTGROUPCONFIG});
    if (this.configObject.id !== '') {
      configObject.id = this.configObject.id;
    }

    const crawlHostGroupConfig = new CrawlHostGroupConfig();
    crawlHostGroupConfig.ipRangeList = formModel.ipRangeList
      .map(ipRange => new IpRange({ipFrom: ipRange.ipFrom, ipTo: ipRange.ipTo}));
    crawlHostGroupConfig.minTimeBetweenPageLoadMs = parseInt(formModel.minTimeBetweenPageLoadMs, 10) || 0;
    crawlHostGroupConfig.maxTimeBetweenPageLoadMs = parseInt(formModel.maxTimeBetweenPageLoadMs, 10) || 0;
    crawlHostGroupConfig.delayFactor = parseFloat(formModel.delayFactor) || 0;
    crawlHostGroupConfig.maxRetries = parseInt(formModel.maxRetries, 10) || 0;
    crawlHostGroupConfig.retryDelaySeconds = parseInt(formModel.retryDelaySeconds, 10) || 0;

    configObject.meta = formModel.meta;
    configObject.crawlHostGroupConfig = crawlHostGroupConfig;

    return configObject;

  }

  private initIpRange(ipRange = new IpRange()) {
    return this.fb.group({
      ipFrom: [ipRange.ipFrom || '', [CrawlHostGroupConfigIpValidation.ipAddressValidator]],
      ipTo: [ipRange.ipTo || '', [CrawlHostGroupConfigIpValidation.ipAddressValidator]],
    });
  }

  private allIpRangesValid(): boolean {
    return this.ipRangeControlArray.controls.every(control =>
      CrawlHostGroupConfigIpValidation.isValidRange(
        control.get('ipFrom').value,
        control.get('ipTo').value,
      )
    );
  }
}
