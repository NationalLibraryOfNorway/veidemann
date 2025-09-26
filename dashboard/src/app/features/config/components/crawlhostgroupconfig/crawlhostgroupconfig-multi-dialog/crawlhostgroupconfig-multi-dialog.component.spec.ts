import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawlHostGroupConfigMultiDialogComponent} from './crawlhostgroupconfig-multi-dialog.component';
import {UntypedFormBuilder} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {LabelService} from '../../../services';
import {AuthService} from '../../../../../core';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {of} from 'rxjs';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlHostGroupConfigMultiDialogComponent', () => {
  let component: CrawlHostGroupConfigMultiDialogComponent;
  let fixture: ComponentFixture<CrawlHostGroupConfigMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLHOSTGROUPCONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [UntypedFormBuilder,
        ...provideCoreTesting,
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
        {provide: AuthService, useValue: {canUpdate: () => true}},
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ],
      declarations: [CrawlHostGroupConfigMultiDialogComponent, LabelMultiComponent, DurationPickerComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlHostGroupConfigMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
