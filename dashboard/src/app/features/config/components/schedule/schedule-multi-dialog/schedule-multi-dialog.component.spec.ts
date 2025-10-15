import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ScheduleMultiDialogComponent} from './schedule-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';

describe('ScheduleMultiDialogComponent', () => {
  let component: ScheduleMultiDialogComponent;
  let fixture: ComponentFixture<ScheduleMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLSCHEDULECONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ScheduleMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MatDialogRef, useValue: {}},
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
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
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
