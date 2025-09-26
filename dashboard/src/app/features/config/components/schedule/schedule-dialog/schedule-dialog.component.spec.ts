import { ScheduleDialogComponent } from './schedule-dialog.component';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { UntypedFormBuilder } from '@angular/forms';
import { ConfigObject, Kind } from '../../../../../shared/models';
import { AnnotationComponent, LabelComponent, MetaComponent } from '../..';
import { LabelService } from '../../../services';
import { of } from 'rxjs';
import { AuthService } from '../../../../../core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

describe('ScheduleDialogComponent', () => {
  let fixture: ComponentFixture<ScheduleDialogComponent>;
  let component: ScheduleDialogComponent;


  beforeEach(() => {
      TestBed.configureTestingModule({
        imports: [
        ],
        declarations: [MetaComponent, LabelComponent, AnnotationComponent],
        providers: [
          UntypedFormBuilder,
          MatDialogModule,
          {
            provide: LabelService,
            useValue: {
              getLabelKeys: () => of([])
            }
          },
          {
            provide: MAT_DIALOG_DATA, useValue: {
              configObject: new ConfigObject({ kind: Kind.CRAWLSCHEDULECONFIG }),
              options: {}
            }
          },
          { provide: MatDialogRef, useValue: {} },
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
