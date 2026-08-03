import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlHostGroupConfigDialogComponent} from './crawlhostgroupconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
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
      imports: [CrawlHostGroupConfigDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ],
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
