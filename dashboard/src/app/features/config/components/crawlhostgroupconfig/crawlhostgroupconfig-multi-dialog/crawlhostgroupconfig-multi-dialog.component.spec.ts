import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawlHostGroupConfigMultiDialogComponent} from './crawlhostgroupconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
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
      imports: [CrawlHostGroupConfigMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ],
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
