import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlHostGroupConfigDialogComponent} from './crawlhostgroupconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {MetaComponent} from '../../meta/meta.component';
import {LabelComponent} from '../../label/label.component';
import {LabelService} from '../../../services';
import {of} from 'rxjs';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {AnnotationComponent} from '../../annotation/annotation.component';
import {AuthService} from '../../../../../core';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlHostGroupConfigDialogComponent', () => {
  let component: CrawlHostGroupConfigDialogComponent;
  let fixture: ComponentFixture<CrawlHostGroupConfigDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLHOSTGROUPCONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [
        ...provideCoreTesting,
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
        {provide: AuthService, useValue: {canUpdate: () => true}},
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ],
      declarations: [CrawlHostGroupConfigDialogComponent, MetaComponent, LabelComponent, AnnotationComponent, DurationPickerComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlHostGroupConfigDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
