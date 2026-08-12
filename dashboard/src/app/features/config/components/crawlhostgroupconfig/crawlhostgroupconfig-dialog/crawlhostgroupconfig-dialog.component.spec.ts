import {ComponentFixture, TestBed} from '@angular/core/testing';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatFormFieldHarness} from '@angular/material/form-field/testing';
import {MatInputHarness} from '@angular/material/input/testing';

import {CrawlHostGroupConfigDialogComponent} from './crawlhostgroupconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlHostGroupConfigDialogComponent', () => {
  let component: CrawlHostGroupConfigDialogComponent;
  let fixture: ComponentFixture<CrawlHostGroupConfigDialogComponent>;
  let loader: HarnessLoader;

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
    loader = TestbedHarnessEnvironment.loader(fixture);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('uses the detail form IP range interaction and field nomenclature', async () => {
    const addIpRangeChip = fixture.nativeElement.querySelector(
      'mat-chip[data-testid="addIpRangeButton"]'
    ) as HTMLElement;

    expect(addIpRangeChip).not.toBeNull();
    expect(addIpRangeChip.textContent).toContain('Add IP range');

    addIpRangeChip.click();
    await fixture.whenStable();

    const fromField = await loader.getHarness(
      MatFormFieldHarness.with({selector: '[data-testid="ipRangeFrom"]'})
    );
    const toField = await loader.getHarness(
      MatFormFieldHarness.with({selector: '[data-testid="ipRangeTo"]'})
    );

    expect(await fromField.getLabel()).toBe('From IP address');
    expect(await toField.getLabel()).toBe('To IP address');
    expect(await fromField.getTextHints()).toEqual(['From']);
    expect(await toField.getTextHints()).toEqual(['To']);
    expect(fixture.nativeElement.querySelectorAll('mat-error')).toHaveLength(0);

    const fromInput = await fromField.getControl() as MatInputHarness;
    await fromInput.setValue('not-an-ip-address');
    await fromInput.blur();
    await fixture.whenStable();

    expect(await fromField.getTextErrors()).toEqual(['A valid IP address is required.']);
    expect(
      fixture.nativeElement.querySelector('[data-testid="ipRangeFrom"]')
        .querySelector('.mat-mdc-form-field-subscript-dynamic-size')
    ).not.toBeNull();
  });
});
