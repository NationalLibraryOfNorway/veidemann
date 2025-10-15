import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlConfigMultiDialogComponent} from './crawlconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlConfigMultiDialogComponent', () => {
  let component: CrawlConfigMultiDialogComponent;
  let fixture: ComponentFixture<CrawlConfigMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLCONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlConfigMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlConfigMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
