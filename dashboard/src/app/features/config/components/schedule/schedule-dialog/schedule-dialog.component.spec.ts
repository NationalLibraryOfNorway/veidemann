import {ScheduleDialogComponent} from './schedule-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AuthService} from '../../../../../core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';

describe('ScheduleDialogComponent', () => {
  let fixture: ComponentFixture<ScheduleDialogComponent>;
  let component: ScheduleDialogComponent;


  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        ScheduleDialogComponent
      ],
      providers: [
        ...provideCoreTesting,
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
        {
          provide: MAT_DIALOG_DATA, useValue: {
            configObject: new ConfigObject({kind: Kind.CRAWLSCHEDULECONFIG}),
            options: {}
          }
        },
        {provide: MatDialogRef, useValue: {}},
        {
          provide: AuthService, useValue: {
            canUpdate: () => true,
            canEdit: () => true
          }
        }
      ]
    });

    fixture = TestBed.createComponent(ScheduleDialogComponent);
    component = fixture.componentInstance;
  });


  it('should create', () => {
    expect(component).toBeTruthy();
  });

});
