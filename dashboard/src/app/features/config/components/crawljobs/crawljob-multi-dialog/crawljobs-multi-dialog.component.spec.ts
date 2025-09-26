import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlJobMultiDialogComponent} from './crawljobs-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {FilesizeInputComponent} from '../../filesize-input/filesize-input.component';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {UntypedFormBuilder} from '@angular/forms';
import {LabelService} from '../../../services';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlJobMultiDialogComponent', () => {
  let component: CrawlJobMultiDialogComponent;
  let fixture: ComponentFixture<CrawlJobMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLJOB}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      declarations: [CrawlJobMultiDialogComponent,
        FilesizeInputComponent,
        DurationPickerComponent,
        LabelMultiComponent],
      providers: [
        ...provideCoreTesting,
        UntypedFormBuilder,
        {provide: LabelService, useValue: {}},
        {provide: AuthService, useValue: {canUpdate: () => true}},
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlJobMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
