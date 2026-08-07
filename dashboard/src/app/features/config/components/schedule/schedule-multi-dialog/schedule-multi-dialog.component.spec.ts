import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ScheduleMultiDialogComponent} from './schedule-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';
import {MatDateRangeInputHarness} from '@angular/material/datepicker/testing';

describe('ScheduleMultiDialogComponent', () => {
  let component: ScheduleMultiDialogComponent;
  let fixture: ComponentFixture<ScheduleMultiDialogComponent>;
  let loader: HarnessLoader;

  let dialogData: ConfigDialogData;

  beforeEach(() => {
    dialogData = {
      configObject: new ConfigObject({
        kind: Kind.CRAWLSCHEDULECONFIG,
        crawlScheduleConfig: {
          cronExpression: '',
          validFrom: '2026-07-01T00:00:00.000Z',
          validTo: '2026-07-31T23:59:59.999Z',
        },
      }),
      options: {},
      allSelected: false,
    };
    TestBed.configureTestingModule({
      imports: [ScheduleMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialogRef, useValue: {}},
        {provide: MAT_DIALOG_DATA, useFactory: () => dialogData},
        {
          provide: DateAdapter,
          useClass: DateFnsAdapter,
          deps: [MAT_DATE_LOCALE]
        },
        {
          provide: MAT_DATE_FORMATS,
          useValue: MAT_DATE_FNS_FORMATS
        },
        {
          provide: MAT_DATE_LOCALE,
          useValue: nb,
        },
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ScheduleMultiDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
    loader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the validity period as one inclusive date range field', async () => {
    const range = await loader.getHarness(MatDateRangeInputHarness);
    const startInput = await range.getStartInput();
    const endInput = await range.getEndInput();

    expect(await range.getLabel()).toBe('Validity period');
    expect(await startInput.getPlaceholder()).toBe('Valid from');
    expect(await endInput.getPlaceholder()).toBe('Valid to');
    expect(fixture.nativeElement.querySelectorAll('mat-date-range-input')).toHaveLength(1);
    expect(fixture.nativeElement.querySelectorAll('mat-datepicker-toggle')).toHaveLength(1);
    expect(component.validFrom.value).toBeInstanceOf(Date);
    expect(component.validTo.value).toBeInstanceOf(Date);
  });

  it('serializes both inclusive boundaries in UTC', () => {
    component.form.patchValue({
      validFrom: new Date(2026, 6, 14),
      validTo: new Date(2026, 6, 20),
    });
    component.validFrom.markAsDirty();
    component.validTo.markAsDirty();

    const result = component.onDialogClose();

    expect(result.pathList).toEqual([
      'crawlScheduleConfig.validFrom',
      'crawlScheduleConfig.validTo',
    ]);
    expect(result.updateTemplate.crawlScheduleConfig.validFrom).toBe('2026-07-14T00:00:00.000Z');
    expect(result.updateTemplate.crawlScheduleConfig.validTo).toBe('2026-07-20T23:59:59.999Z');
  });

  it('updates only the edited side of an open-ended range', () => {
    component.form.patchValue({validFrom: new Date(2026, 7, 1), validTo: null});
    component.validFrom.markAsDirty();

    const result = component.onDialogClose();

    expect(result.pathList).toEqual(['crawlScheduleConfig.validFrom']);
    expect(result.updateTemplate.crawlScheduleConfig.validFrom).toBe('2026-08-01T00:00:00.000Z');
    expect(result.updateTemplate.crawlScheduleConfig.validTo).toBe('');
  });

  it('clears one boundary without changing the other boundary', () => {
    component.validTo.setValue(null);
    component.validTo.markAsDirty();

    const result = component.onDialogClose();

    expect(result.pathList).toEqual(['crawlScheduleConfig.validTo']);
    expect(result.updateTemplate.crawlScheduleConfig.validTo).toBe('');
  });

  it('rejects a reversed populated range', () => {
    component.form.patchValue({
      validFrom: new Date(2026, 6, 20),
      validTo: new Date(2026, 6, 14),
    });
    component.validFrom.updateValueAndValidity();
    component.validTo.updateValueAndValidity();
    fixture.detectChanges();

    expect(component.form.invalid).toBe(true);
    expect(component.canUpdate).toBe(false);
  });

  it('clears both range boundaries with backend-safe empty values', () => {
    component.onRemoveValidFromTo();

    const result = component.onDialogClose();

    expect(result.pathList).toEqual([
      'crawlScheduleConfig.validFrom',
      'crawlScheduleConfig.validTo',
    ]);
    expect(result.updateTemplate.crawlScheduleConfig.validFrom).toBe('');
    expect(result.updateTemplate.crawlScheduleConfig.validTo).toBe('');
  });

  it('uses a newly selected range after clearing', () => {
    component.onRemoveValidFromTo();
    component.form.patchValue({
      validFrom: new Date(2026, 8, 1),
      validTo: new Date(2026, 8, 30),
    });
    component.validFrom.markAsDirty();
    component.validTo.markAsDirty();
    component.onValidityRangeChanged();

    const result = component.onDialogClose();

    expect(result.updateTemplate.crawlScheduleConfig.validFrom).toBe('2026-09-01T00:00:00.000Z');
    expect(result.updateTemplate.crawlScheduleConfig.validTo).toBe('2026-09-30T23:59:59.999Z');
  });

  it('leaves mixed range values untouched until the range is activated and edited', () => {
    component.validFrom.disable();
    component.validTo.disable();

    expect(component.onDialogClose().pathList).toEqual([]);

    component.enableValidityRange();
    component.validFrom.setValue(new Date(2026, 9, 1));
    component.validFrom.markAsDirty();

    expect(component.onDialogClose().pathList).toEqual(['crawlScheduleConfig.validFrom']);
  });
});
