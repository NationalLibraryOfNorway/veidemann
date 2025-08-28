import { ScheduleDialogComponent } from './schedule-dialog.component';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { UntypedFormBuilder } from '@angular/forms';
import { CoreTestingModule } from '../../../../core/core.testing.module';
import { ConfigObject, Kind } from '../../../../shared/models';
import { CommonsModule } from '../../../../commons';
import { AnnotationComponent, LabelComponent, MetaComponent } from '../..';
import { ConfigDialogData } from '../../../func';
import { LabelService } from '../../../services';
import { of } from 'rxjs';
import { AuthService } from '../../../../core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

describe('ScheduleDialogComponent', () => {
  let fixture: ComponentFixture<ScheduleDialogComponent>;
  let component: ScheduleDialogComponent;


  beforeEach(() => {
      TestBed.configureTestingModule({
        imports: [
          CoreTestingModule.forRoot(),
          CommonsModule,
        ],
        declarations: [MetaComponent, LabelComponent, AnnotationComponent],
        providers: [
          UntypedFormBuilder,
          CoreTestingModule.forRoot(),
          MatDialogModule,
          CommonsModule,
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
