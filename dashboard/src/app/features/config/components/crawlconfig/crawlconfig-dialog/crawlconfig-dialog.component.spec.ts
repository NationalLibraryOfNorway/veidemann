import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlConfigDialogComponent} from './crawlconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';

describe('CrawlConfigDialogComponent', () => {
  let component: CrawlConfigDialogComponent;
  let fixture: ComponentFixture<CrawlConfigDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLCONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlConfigDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlConfigDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders Create screenshots as a normal checkbox', async () => {
    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    expect(await checkbox.getLabelText()).toBe('Create screenshots');
  });
});
