import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawlHostGroupConfigDetailsComponent} from './crawlhostgroupconfig-details.component';
import {SimpleChange} from '@angular/core';
import {Annotation, ConfigObject, CrawlHostGroupConfig, Kind, Label, Meta} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {By} from '@angular/platform-browser';
import {MatFormFieldHarness} from '@angular/material/form-field/testing';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import { describe, it, expect, beforeEach } from 'vitest';

const exampleCrawlHostGroupConfig: ConfigObject = {
  id: 'configObject_id',
  apiVersion: 'v1',
  kind: Kind.CRAWLHOSTGROUPCONFIG,
  meta: new Meta({
    name: 'Example CrawlHostGroupConfig',
    createdBy: 'test',
    created: '01.01.1970',
    lastModified: '01.01.2021',
    lastModifiedBy: 'test',
    description: 'This is an example CrawlHostGroupConfig',
    labelList: [new Label({ key: 'test', value: 'label' })],
    annotationList: [new Annotation({ key: 'test', value: 'annotation' })]
  }),
  crawlHostGroupConfig: new CrawlHostGroupConfig({
    ipRangeList: [],
    minTimeBetweenPageLoadMs: 1000,
    maxTimeBetweenPageLoadMs: 2000,
    delayFactor: 1.0,
    maxRetries: 3,
    retryDelaySeconds: 3
  })
};

describe('CrawlHostGroupConfigDetailsComponent', () => {
  let component: CrawlHostGroupConfigDetailsComponent;
  let fixture: ComponentFixture<CrawlHostGroupConfigDetailsComponent>;
  let loader: HarnessLoader;

  let saveButton: MatButtonHarness;
  let updateButton: MatButtonHarness;
  let revertButton: MatButtonHarness;
  let deleteButton: MatButtonHarness;
  let addIpRangeButton: MatButtonHarness;
  let removeIpRangeButton: MatButtonHarness;

  let ipRangeListElement: any;
  let delayFactorFormField: MatFormFieldHarness;
  let delayFactorInput: any;
  let ipRangeFromFormField: MatFormFieldHarness;
  let ipRangeFromInput: any;
  let ipRangeToFormField: MatFormFieldHarness;
  let ipRangeToInput: any;


  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CrawlHostGroupConfigDetailsComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlHostGroupConfigDetailsComponent);
    loader = TestbedHarnessEnvironment.loader(fixture);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject(exampleCrawlHostGroupConfig);
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, null)
    });
    await fixture.whenStable();

    delayFactorFormField = await loader.getHarness<MatFormFieldHarness>(MatFormFieldHarness
      .with({ selector: '[data-testid="delayFactor"]' }));
    delayFactorInput = await delayFactorFormField.getControl();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('Creating a new crawlHostGroupConfig', () => {

    beforeEach(async () => {
      component.configObject.id = '';
      component.ngOnChanges({
        configObject: new SimpleChange(null, component.configObject, null)
      });
      await fixture.whenStable();
      saveButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({ text: 'SAVE' }));
    });

    it('show save button when creating a new config if form is valid', async () => {
      expect(await saveButton.isDisabled()).toBeFalsy();
      expect(component.canSave).toBeTruthy();
    });
  });

  describe('Updating a crawlHostGroupConfig', () => {
    beforeEach(async () => {
      updateButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({ text: 'UPDATE' }));
      deleteButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({ text: 'DELETE' }));
      revertButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({ text: 'REVERT' }));
      addIpRangeButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness
        .with({ selector: '[data-testid="addIpRangeButton"]' }));
      ipRangeListElement = fixture.debugElement.query(By.css('[data-testid="ipRangeList"]'));
    });

    it('update button should be active if form is updated and valid', async () => {
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
      await delayFactorInput.setValue('2.0');
      await fixture.whenStable();
      expect(await updateButton.isDisabled()).toBeFalsy();
      expect(component.canUpdate).toBeTruthy();
    });


    it('clicking update button emits an update event', async () => {
      expect(component.canUpdate).toBeFalsy();
      expect(await updateButton.isDisabled()).toBeTruthy();

      let update: ConfigObject | undefined;
      component.update.subscribe((config: ConfigObject) => {
        update = config;
      });

      await delayFactorInput.setValue('2');
      await fixture.whenStable();

      expect(component.canUpdate).toBeTruthy();
      expect(await updateButton.isDisabled()).toBeFalsy();

      await updateButton.click();
      expect(update.crawlHostGroupConfig.delayFactor).toBe(2);
    });

    /** Testing IP-range formgroup */

    it('clicking add IP-range button enables input fields for adding a range', async () => {
      expect(ipRangeListElement).toBeNull();

      await addIpRangeButton.click();
      await fixture.whenStable();

      expect(ipRangeListElement).toBeDefined();
      ipRangeFromFormField = await loader.getHarness<MatFormFieldHarness>(MatFormFieldHarness
        .with({ selector: '[data-testid="ipRangeFrom"]' }));
      ipRangeFromInput = await ipRangeFromFormField.getControl();
      ipRangeToFormField = await loader.getHarness<MatFormFieldHarness>(MatFormFieldHarness
        .with({ selector: '[data-testid="ipRangeTo"]' }));
      ipRangeToInput = await ipRangeToFormField.getControl();

      expect(await ipRangeFromInput.getValue()).toEqual('');
      expect(await ipRangeToInput.getValue()).toEqual('');
    });

    it('clicking remove button removes range from list', async () => {
      expect(ipRangeListElement).toBeNull();
      await addIpRangeButton.click();
      await fixture.whenStable();
      expect(ipRangeListElement).toBeDefined();
      removeIpRangeButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness
        .with({ selector: '[data-testid="removeIpRangeButton"]' }));
      await removeIpRangeButton.click();
      await fixture.whenStable();
      expect(ipRangeListElement).toBeNull();
    });

    it('update button should be disabled if ip range is invalid', async () => {
      await addIpRangeButton.click();
      await fixture.whenStable();

      ipRangeFromFormField = await loader.getHarness<MatFormFieldHarness>(
        MatFormFieldHarness.with({ selector: '[data-testid="ipRangeFrom"]' })
      );
      ipRangeFromInput = await ipRangeFromFormField.getControl();

      ipRangeToFormField = await loader.getHarness<MatFormFieldHarness>(
        MatFormFieldHarness.with({ selector: '[data-testid="ipRangeTo"]' })
      );
      ipRangeToInput = await ipRangeToFormField.getControl();

      // Set invalid range
      await ipRangeFromInput.setValue('192.168.1.1');
      await ipRangeToInput.setValue('193.168.1.100');
      await fixture.whenStable();

      expect(component.canUpdate).toBeFalsy();

      // Query error element
      const invalidRangeError = fixture.debugElement.query(
        By.css('[data-testid="ipRangeInvalidError"]')
      );

      expect(invalidRangeError).not.toBeNull();

      // IMPORTANT: innerText does not exist in jsdom
      const errorText = (invalidRangeError.nativeElement as HTMLElement).textContent?.trim();

      expect(errorText).toBe('The IP range is not valid');

      // Make it valid
      await ipRangeToInput.setValue('192.169.1.100');
      await fixture.whenStable();

      expect(component.canUpdate).toBeTruthy();
    });


    /** Testing revert button */

    it('When form is dirty the revert button becomes active,clicking it reverts form back to initial values',
      async () => {
        expect(await revertButton.isDisabled()).toBeTruthy();
        expect(component.canRevert).toBeFalsy();

        await delayFactorInput.setValue(3.0);
        await fixture.whenStable();
        expect(await revertButton.isDisabled()).toBeFalsy();
        expect(component.canRevert).toBeTruthy();

        await revertButton.click();
        await fixture.whenStable();
        expect(await delayFactorInput.getValue()).toBe('1');
        expect(component.canUpdate).toBeFalsy();
        expect(await revertButton.isDisabled()).toBeTruthy();
      });

    /**  Testing delete button */
    it('Clicking delete button emits a delete event', async () => {
      let del: ConfigObject | undefined;
      component.delete.subscribe((config: ConfigObject) => {
        del = config;
      });

      expect(await deleteButton.isDisabled()).toBeFalsy();
      expect(component.canDelete).toBeTruthy();
      await deleteButton.click();

      expect(del.crawlHostGroupConfig).toBe(component.configObject.crawlHostGroupConfig);
    });
  });
});
