import {ComponentFixture, TestBed} from '@angular/core/testing';
import {EntityDetailsComponent} from './entity-details.component';
import {Annotation, ConfigObject, CrawlEntity, Kind, Label, Meta} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {SimpleChange} from '@angular/core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

const exampleCrawlEntity: ConfigObject = {
  id: 'configObject_id',
  apiVersion: 'v1',
  kind: Kind.CRAWLENTITY,
  meta: new Meta({
    name: 'Example Entity',
    createdBy: 'test',
    created: '01.01.1970',
    lastModified: '01.01.2021',
    lastModifiedBy: 'test',
    description: 'This is an example entity',
    labelList: [new Label({key: 'test', value: 'label'})],
    annotationList: [new Annotation({key: 'test', value: 'annotation'})]
  }),
  crawlEntity: new CrawlEntity()
};

describe('EntityDetailsComponent', () => {
  let component: EntityDetailsComponent;
  let fixture: ComponentFixture<EntityDetailsComponent>;
  let loader: HarnessLoader;

  let saveButton: MatButtonHarness;
  let updateButton: MatButtonHarness;
  let revertButton: MatButtonHarness;
  let deleteButton: MatButtonHarness;


  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        EntityDetailsComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(EntityDetailsComponent);
    loader = TestbedHarnessEnvironment.loader(fixture);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject(exampleCrawlEntity);
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, null)
    });
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('Creating a new crawlEntity', () => {

    beforeEach(async () => {
      component.configObject.id = '';
      component.ngOnChanges({
        configObject: new SimpleChange(null, component.configObject, null)
      });
      await fixture.whenStable();
      saveButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'SAVE'}));
    });

    it('show save button when creating a new config if form is valid', async () => {
      expect(await saveButton.isDisabled()).toBeFalsy();
      expect(component.canSave).toBeTruthy();
    });
  });

  describe('Updating a crawlEntity', () => {
    beforeEach(async () => {
      await fixture.whenStable();
      updateButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'UPDATE'}));
      deleteButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'DELETE'}));
      revertButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'REVERT'}));
    });

    it('update button should be active if form is updated and valid', async () => {
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
    });
  });
});
