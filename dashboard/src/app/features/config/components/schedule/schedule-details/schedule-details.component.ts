import {ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';

import {ConfigObject, CrawlScheduleConfig, Kind, Meta,} from '../../../../../shared/models';


import {
  VALID_CRON_DOM_PATTERN,
  VALID_CRON_DOW_PATTERN,
  VALID_CRON_HOUR_PATTERN,
  VALID_CRON_MINUTE_PATTERN,
  VALID_CRON_MONTH_PATTERN
} from '../../../../../shared/validation';

import {AuthService} from '../../../../../core/auth';
import {DateTime} from '../../../../../shared/func';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInput} from '@angular/material/input';
import {MatListSubheaderCssMatStyler} from '@angular/material/list';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {JsonPipe} from '@angular/common';
import {MatButtonModule} from '@angular/material/button';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE, MatNativeDateModule} from '@angular/material/core';


@Component({
  selector: 'app-schedule-details',
  templateUrl: './schedule-details.component.html',
  styleUrls: ['./schedule-details.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FlexLayoutModule,
    JsonPipe,
    MatButtonModule,
    MatCardModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatListSubheaderCssMatStyler,
    MetaComponent,
    ReactiveFormsModule,
    MatNativeDateModule,
  ],
  providers: [
    { provide: MAT_DATE_LOCALE, useValue: 'nb' },
    { provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE] },
    { provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS }
  ],
  standalone: true
})
export class ScheduleDetailsComponent implements OnChanges {
  @Input()
  configObject: ConfigObject;

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  update = new EventEmitter<ConfigObject>();

  @Output()
  delete = new EventEmitter<ConfigObject>();

  form: UntypedFormGroup;

  constructor(protected fb: UntypedFormBuilder,
              protected authService: AuthService) {
    this.createForm();
  }

  protected static setCronExpression(cronExpression): string {
    const {minute, hour, dom, month, dow} = cronExpression;
    return minute + ' ' + hour + ' ' + dom + ' ' + month + ' ' + dow;
  }

  get canDelete(): boolean {
    return this.authService.canDelete(this.configObject.kind);
  }

  get canEdit(): boolean {
    return this.authService.canUpdate(this.configObject.kind);
  }

  get showSave(): boolean {
    return this.configObject && !this.configObject.id;
  }

  get canSave(): boolean {
    return this.form.valid && this.canEdit;
  }

  get canUpdate() {
    return this.form.valid && this.form.dirty && this.canEdit;
  }

  get canRevert() {
    return this.canEdit && this.form.dirty;
  }

  get cronExpression() {
    return this.form.get('cronExpression');
  }

  get validFrom() {
    return this.form.get('validFrom');
  }

  get validTo() {
    return this.form.get('validTo');
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['configObject']) {
      if (this.configObject) {
        this.updateForm();
      } else {
        this.form.reset();
      }
    }
  }

  onSave() {
    this.save.emit(this.prepareSave());
  }

  onUpdate(): void {
    this.update.emit(this.prepareSave());
  }

  onDelete(): void {
    this.delete.emit(this.configObject);
  }

  onRevert() {
    this.updateForm();
  }

  protected createForm() {
    this.form = this.fb.group({
      id: '',
      validFrom: '',
      validTo: '',
      cronExpression: this.fb.group({
        minute: ['', [Validators.required, Validators.pattern(VALID_CRON_MINUTE_PATTERN)]],
        hour: ['', [Validators.required, Validators.pattern(VALID_CRON_HOUR_PATTERN)]],
        dom: ['', [Validators.required, Validators.pattern(VALID_CRON_DOM_PATTERN)]],
        month: ['', [Validators.required, Validators.pattern(VALID_CRON_MONTH_PATTERN)]],
        dow: ['', [Validators.required, Validators.pattern(VALID_CRON_DOW_PATTERN)]],
      }),
      meta: new Meta(),
    });
  }

  protected updateForm() {
    const [minute, hour, dom, month, dow] = this.configObject.crawlScheduleConfig.cronExpression.split(' ');
    const cronExpression = {
      minute: minute || '',
      hour: hour || '',
      dom: dom || '',
      month: month || '',
      dow: dow || '',
    };
    const validFrom = this.configObject.crawlScheduleConfig.validFrom
      ? DateTime.adjustTime(this.configObject.crawlScheduleConfig.validFrom)
      : '';
    const validTo = this.configObject.crawlScheduleConfig.validTo
      ? DateTime.adjustTime(this.configObject.crawlScheduleConfig.validTo)
      : '';

    this.form.setValue({
      id: this.configObject.id,
      meta: this.configObject.meta,
      cronExpression: cronExpression,
      validFrom: validFrom,
      validTo: validTo,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareSave(): ConfigObject {
    const formModel = this.form.value;

    const configObject = new ConfigObject({kind: Kind.CRAWLSCHEDULECONFIG});
    if (this.configObject.id !== '') {
      configObject.id = this.configObject.id;
    }

    const crawlScheduleConfig = new CrawlScheduleConfig();
    const validFromUTC = DateTime.dateToUtc(formModel.validFrom, true);
    const validToUTC = DateTime.dateToUtc(formModel.validTo, false);

    crawlScheduleConfig.validFrom = validFromUTC ? validFromUTC : null;
    crawlScheduleConfig.validTo = validToUTC ? validToUTC : null;
    crawlScheduleConfig.cronExpression = ScheduleDetailsComponent.setCronExpression(this.form.value.cronExpression);

    configObject.meta = formModel.meta;
    configObject.crawlScheduleConfig = crawlScheduleConfig;
    return configObject;
  }
}
