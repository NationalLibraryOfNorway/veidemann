import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawlJobDetailsComponent} from './crawl-job-details.component';
import {
  Annotation,
  BrowserScript,
  ConfigObject,
  ConfigRef,
  CrawlConfig,
  CrawlJob,
  CrawlScheduleConfig,
  Kind,
  Label,
  Meta
} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {CrawlLimitsConfig} from '../../../../../shared/models/config/crawljob.model';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {SimpleChange} from '@angular/core';
import {MatChipOptionHarness} from '@angular/material/chips/testing';
import {ExtraConfig} from '../../../../../shared/models/config/crawlconfig.model';
import {MatSelectHarness} from '@angular/material/select/testing';
import {provideCoreTesting} from '../../../../../core/core.testing.module';


const exampleCrawlConfigs: ConfigObject[] = [
  new ConfigObject({
    id: 'configObject_id',
    apiVersion: 'v1',
    kind: Kind.CRAWLCONFIG,
    meta: new Meta({
      name: 'Example CrawlConfig',
      createdBy: 'test',
      created: '01.01.1970',
      lastModified: '01.01.2021',
      lastModifiedBy: 'test',
      description: 'This is an example CrawlConfig',
      labelList: [new Label({key: 'test', value: 'label'})],
      annotationList: [new Annotation({key: 'test', value: 'annotation'})]
    }),
    crawlConfig: new CrawlConfig({
      priorityWeight: 100,
      minimumDnsTtlS: 100,
      collectionRef: null,
      politenessRef: null,
      browserConfigRef: null,
      extra: new ExtraConfig({createScreenshot: true}),
    })
  })
];

const exampleBrowserScripts: ConfigObject[] = [
  new ConfigObject({
    id: 'configObject_id',
    apiVersion: 'v1',
    kind: Kind.BROWSERSCRIPT,
    meta: new Meta({
      name: 'Example BrowserScript',
      createdBy: 'test',
      created: '01.01.1970',
      lastModified: '01.01.2021',
      lastModifiedBy: 'test',
      description: 'This is an example BrowserScript',
      labelList: [new Label({key: 'test', value: 'label'})],
      annotationList: [new Annotation({key: 'test', value: 'annotation'})]
    }),
    browserScript: new BrowserScript({
      script: 'console.log(\'test\')',
      urlRegexpList: [],
      browserScriptType: null
    })
  })
];

const exampleSchedules: ConfigObject[] = [
  new ConfigObject({
    id: 'configObject_id',
    apiVersion: 'v1',
    kind: Kind.CRAWLSCHEDULECONFIG,
    meta: new Meta({
      name: 'Example Schedule',
      createdBy: 'test',
      created: '01.01.1970',
      lastModified: '01.01.2021',
      lastModifiedBy: 'test',
      description: 'This is an example Schedule',
      labelList: [new Label({key: 'test', value: 'label'})],
      annotationList: [new Annotation({key: 'test', value: 'annotation'})]
    }),
    crawlScheduleConfig: new CrawlScheduleConfig({
      cronExpression: '*****',
      validFrom: '01.01.1970',
      validTo: '01.01.2021'
    })
  })
];

const exampleCrawlJob: ConfigObject = {
  id: 'configObject_id',
  apiVersion: 'v1',
  kind: Kind.CRAWLJOB,
  meta: new Meta({
    name: 'Example CrawlJob',
    createdBy: 'test',
    created: '01.01.1970',
    lastModified: '01.01.2021',
    lastModifiedBy: 'test',
    description: 'This is an example CrawlJob',
    labelList: [new Label({key: 'test', value: 'label'})],
    annotationList: [new Annotation({key: 'test', value: 'annotation'})]
  }),
  crawlJob: new CrawlJob({
    scheduleRef: new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG, id: 'configObject_id'}),
    crawlConfigRef: new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'configObject_id'}),
    scopeScriptRef: new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'configObject_id'}),
    limits: new CrawlLimitsConfig({
      maxDurationS: 1,
      maxBytes: 1024
    }),
    disabled: false
  })
};

describe('CrawljobDetailsComponent', () => {
  let component: CrawlJobDetailsComponent;
  let fixture: ComponentFixture<CrawlJobDetailsComponent>;
  let loader: HarnessLoader;

  let saveButton: MatButtonHarness;
  let updateButton: MatButtonHarness;
  let revertButton: MatButtonHarness;

  let disabledChip: MatChipOptionHarness;
  let crawlConfigSelect: MatSelectHarness;
  let scheduleSelect: MatSelectHarness;
  let scopeScriptSelect: MatSelectHarness;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CrawlJobDetailsComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlJobDetailsComponent);
    loader = TestbedHarnessEnvironment.loader(fixture);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject(exampleCrawlJob);
    component.scopeScripts = exampleBrowserScripts;
    component.crawlScheduleConfigs = exampleSchedules;
    component.crawlConfigs = exampleCrawlConfigs;
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, null)
    });
    await fixture.whenStable();
    disabledChip = await loader.getHarness(
      MatChipOptionHarness.with({selector: 'app-boolean-state-chip mat-chip-option'}),
    );
    scheduleSelect = await loader.getHarness<MatSelectHarness>(MatSelectHarness
      .with({selector: '[data-testid="scheduleRef"]'}));
    crawlConfigSelect = await loader.getHarness<MatSelectHarness>(MatSelectHarness
      .with({selector: '[data-testid="crawlConfigRef"]'}));
    scopeScriptSelect = await loader.getHarness<MatSelectHarness>(MatSelectHarness
      .with({selector: '[data-testid="scopeScriptRef"]'}));
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('offers a copy button for the saved ID field', () => {
    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
  });

  it('places the deactivated control after metadata in document order', () => {
    const metadata = fixture.nativeElement.querySelector('app-meta') as HTMLElement;
    const deactivated = fixture.nativeElement.querySelector('app-boolean-state-chip') as HTMLElement;

    expect(metadata.compareDocumentPosition(deactivated) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  describe('Creating a new crawlJob', () => {
    beforeEach(async () => {
      component.configObject.id = '';
      component.configObject.crawlJob.crawlConfigRef = new ConfigRef({});
      component.ngOnChanges({
        configObject: new SimpleChange(null, component.configObject, null)
      });
      await fixture.whenStable();
      saveButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'SAVE'}));
    });

    it('show save button when creating a new config if form is valid', async () => {
      expect(await saveButton.isDisabled()).toBeTruthy();
      expect(component.canSave).toBeFalsy();
      await crawlConfigSelect.open();
      const crawlConfigSelectOptions = await crawlConfigSelect.getOptions();
      await crawlConfigSelectOptions[0].click();
      await crawlConfigSelect.close();
      expect(await saveButton.isDisabled()).toBeFalsy();
      expect(component.canSave).toBeTruthy();
    });
  });
  describe('updating a crawlJob', () => {
    beforeEach(async () => {
      await fixture.whenStable();
      updateButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'UPDATE'}));
      revertButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'REVERT'}));
    });

    it('update button should be active if form is updated and valid', async () => {
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
      expect(await disabledChip.isSelected()).toBeFalsy();
      await disabledChip.select();
      await fixture.whenStable();
      expect(await updateButton.isDisabled()).toBeFalsy();
      expect(component.canUpdate).toBeTruthy();
    });

    it('update button should be active if required fields is filled', async () => {
      component.configObject.crawlJob.crawlConfigRef = new ConfigRef({});
      component.configObject.crawlJob.scopeScriptRef = new ConfigRef({});
      component.ngOnChanges({
        configObject: new SimpleChange(null, component.configObject, null)
      });
      await fixture.whenStable();
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
      await crawlConfigSelect.open();
      const crawlConfigSelectOptions = await crawlConfigSelect.getOptions();
      await crawlConfigSelectOptions[0].click();
      await fixture.whenStable();
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
      await scopeScriptSelect.open();
      const scopeScriptSelectOptions = await scopeScriptSelect.getOptions();
      await scopeScriptSelectOptions[1].click();
      await fixture.whenStable();
      expect(await updateButton.isDisabled()).toBeFalsy();
      expect(component.canUpdate).toBeTruthy();
    });

    it('schedule dropdown contains all options', async () => {
      await scheduleSelect.open();
      const options = await scheduleSelect.getOptions();
      expect(options.length).toBe(2);
    });

    it('crawlConfig dropdown contains all options', async () => {
      await crawlConfigSelect.open();
      const options = await crawlConfigSelect.getOptions();
      expect(options.length).toBe(1);
    });

    it('scope script dropdown contains all options', async () => {
      await scopeScriptSelect.open();
      const options = await scopeScriptSelect.getOptions();
      expect(options.length).toBe(2);
    });

    it('clicking update button emits an update event', async () => {
      let update: ConfigObject | undefined;
      component.update.subscribe((config: ConfigObject) => {
        update = config;
      });

      await disabledChip.select();
      await fixture.whenStable();

      await updateButton.click();
      expect(update.crawlJob.disabled).toBe(true);
    });


    it('clicking revert buttons reverts form back to initial values', async () => {
      expect(await revertButton.isDisabled()).toBeTruthy();
      await disabledChip.select();
      await fixture.whenStable();
      expect(component.canRevert).toBeTruthy();
      await revertButton.click();
      await fixture.whenStable();
      expect(await disabledChip.isSelected()).toBe(false);
      expect(component.canRevert).toBeFalsy();
      expect(component.canUpdate).toBeFalsy();
    });

    it('does not render delete in the card actions', async () => {
      expect(await loader.getHarnessOrNull(MatButtonHarness.with({text: 'DELETE'}))).toBeNull();
    });

    it('does not duplicate the run crawl action inside the details card', async () => {
      const runCrawlButtons = await loader.getAllHarnesses<MatButtonHarness>(
        MatButtonHarness.with({text: 'RUN CRAWL'}),
      );

      expect(runCrawlButtons).toHaveLength(0);
    });
  });
});
