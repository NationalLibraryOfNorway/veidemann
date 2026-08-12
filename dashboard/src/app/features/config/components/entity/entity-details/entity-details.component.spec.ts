import {ComponentFixture, TestBed} from '@angular/core/testing';
import {EntityDetailsComponent} from './entity-details.component';
import {
  Annotation,
  ConfigObject,
  CrawlEntity,
  Kind,
  Label,
  Meta,
} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {SimpleChange} from '@angular/core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {provideRouter} from '@angular/router';
import {AuthService} from '../../../../../core';

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


  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        EntityDetailsComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {
          provide: AuthService,
          useValue: {
            canCreate: () => true,
            canUpdate: () => true,
            canDelete: () => true,
            canRead: () => true,
            canRunCrawl: () => true,
          },
        },
        provideRouter([]),
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

  it('renders editor fields directly without a duplicated card header', () => {
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-card-header')).toBeNull();
    expect(fixture.nativeElement.querySelector('h1')).toBeNull();
    expect(fixture.nativeElement.querySelector('.entity-details-fields app-meta')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.config-form-actions')).not.toBeNull();
  });

  it('does not duplicate the page title for an unsaved configuration', () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLENTITY, meta: new Meta()});
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, false)
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-title')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('New (unsaved)');
  });

  it('offers a copy button for the saved ID field', () => {
    const fields = fixture.nativeElement.querySelector('.entity-details-fields') as HTMLElement;
    const idField = fixture.nativeElement.querySelector('mat-form-field:has([formControlName="id"])') as HTMLElement;

    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
    expect(fields.classList).toContain('layout-column');
    expect(idField.parentElement).toBe(fields);
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
    });

    it('update button should be active if form is updated and valid', async () => {
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
    });
  });
});
