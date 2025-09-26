import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlJobDialogComponent} from './crawljob-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {
  AnnotationComponent,
  DurationPickerComponent,
  FilesizeInputComponent,
  LabelComponent,
  MetaComponent
} from '../..';
import {LabelService} from '../../../services';
import {of} from 'rxjs';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlJobDialogComponent', () => {
  let component: CrawlJobDialogComponent;
  let fixture: ComponentFixture<CrawlJobDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLJOB}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      declarations: [
        MetaComponent,
        CrawlJobDialogComponent,
        FilesizeInputComponent,
        DurationPickerComponent,
        LabelComponent,
        AnnotationComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
        {
          provide: AuthService,
          useValue: {
            canUpdate: () => true,
          }
        },
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlJobDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
