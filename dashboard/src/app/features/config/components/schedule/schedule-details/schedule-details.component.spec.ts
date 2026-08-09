import {SimpleChange} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ConfigObject, CrawlScheduleConfig, Kind, Meta} from '../../../../../shared/models';
import {ScheduleDetailsComponent} from './schedule-details.component';

describe('ScheduleDetailsComponent', () => {
  let fixture: ComponentFixture<ScheduleDetailsComponent>;
  let component: ScheduleDetailsComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ScheduleDetailsComponent],
      providers: [
        ...provideCoreTesting,
        {provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE]},
        {provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS},
        {provide: MAT_DATE_LOCALE, useValue: nb},
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(ScheduleDetailsComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({
      id: 'schedule-1', kind: Kind.CRAWLSCHEDULECONFIG, meta: new Meta({name: 'Schedule'}),
      crawlScheduleConfig: new CrawlScheduleConfig({cronExpression: '0 6 * * *'}),
    });
    component.ngOnChanges({configObject: new SimpleChange(null, component.configObject, true)});
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('uses one Material date-range input and keeps the remaining schedule fields', () => {
    expect(component).toBeTruthy();
    expect(fixture.nativeElement.querySelectorAll('mat-date-range-input')).toHaveLength(1);
    expect(fixture.nativeElement.querySelectorAll('[matstartdate]')).toHaveLength(1);
    expect(fixture.nativeElement.querySelectorAll('[matenddate]')).toHaveLength(1);
    expect(fixture.nativeElement.querySelectorAll('.cron-expression input')).toHaveLength(5);
    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
  });

  it('serializes the range to inclusive UTC day boundaries', () => {
    component.validFrom.setValue(new Date(2026, 6, 14));
    component.validTo.setValue(new Date(2026, 6, 20));
    const saved = component['prepareSave']();
    expect(saved.crawlScheduleConfig.validFrom).toBe('2026-07-14T00:00:00.000Z');
    expect(saved.crawlScheduleConfig.validTo).toBe('2026-07-20T23:59:59.999Z');
  });

  it('rejects a reversed range', () => {
    component.validFrom.setValue(new Date(2026, 6, 20));
    component.validTo.setValue(new Date(2026, 6, 14));
    fixture.detectChanges();
    expect(component.form.invalid).toBe(true);
    expect(component.validFrom.hasError('matStartDateInvalid') || component.validTo.hasError('matEndDateInvalid')).toBe(true);
  });

  it('validates cron fields and emits updated cron expressions', () => {
    component.cronExpression.get('minute').setValue('60');
    expect(component.form.invalid).toBe(true);
    component.cronExpression.get('minute').setValue('10');
    expect(component.form.valid).toBe(true);
    let update: ConfigObject | undefined;
    component.update.subscribe(value => update = value);
    component.onUpdate();
    expect(update?.crawlScheduleConfig.cronExpression).toBe('10 6 * * *');
  });

  it('reverts range and cron changes', () => {
    component.validFrom.setValue(new Date(2026, 0, 1));
    component.cronExpression.get('minute').setValue('10');
    component.onRevert();
    expect(component.validFrom.value).toBe('');
    expect(component.cronExpression.get('minute').value).toBe('0');
    expect(component.form.pristine).toBe(true);
  });
});
