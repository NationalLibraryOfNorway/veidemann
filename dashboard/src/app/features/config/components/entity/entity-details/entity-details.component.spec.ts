import {ComponentFixture, TestBed} from '@angular/core/testing';
import {EntityDetailsComponent} from './entity-details.component';
import {Annotation, ConfigObject, CrawlEntity, Kind, Label, ListDataSource, Meta} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {SimpleChange} from '@angular/core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {provideRouter} from '@angular/router';
import {EMPTY, of} from 'rxjs';
import {ConfigQuery} from '../../../../../shared/func';
import {MatChipOptionHarness} from '@angular/material/chips/testing';

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

  it('renders the configuration metadata and kind icon in the card header', () => {
    fixture.detectChanges();

    const card = fixture.nativeElement.querySelector('mat-card') as HTMLElement;
    const header = fixture.nativeElement.querySelector('mat-card-header') as HTMLElement;
    const avatar = header.querySelector('div[mat-card-avatar]') as HTMLElement;
    expect(card.getAttribute('appearance')).toBe('outlined');
    expect(card.classList).toContain('mat-mdc-card-outlined');
    expect(avatar.querySelector('mat-icon').textContent.trim()).toBe('business');
    expect(header.querySelector('mat-card-title').textContent.trim()).toBe('Example Entity');
    expect(header.querySelector('mat-card-subtitle').textContent.trim()).toBe('This is an example entity');
    expect(header.querySelector('mat-card-subtitle').classList).toContain('config-card-description');
  });

  it('uses the unsaved title fallback and omits an empty description', () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLENTITY, meta: new Meta()});
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, false)
    });
    fixture.detectChanges();

    const header = fixture.nativeElement.querySelector('mat-card-header') as HTMLElement;
    expect(header.querySelector('mat-card-title').textContent.trim()).toBe('New (unsaved)');
    expect(header.querySelector('mat-card-subtitle')).toBeNull();
  });

  it('offers a copy button for the saved ID field', () => {
    const fields = fixture.nativeElement.querySelector('.entity-details-fields') as HTMLElement;
    const idField = fixture.nativeElement.querySelector('mat-form-field:has([formControlName="id"])') as HTMLElement;

    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
    expect(fields.classList).toContain('layout-column');
    expect(idField.parentElement).toBe(fields);
  });

  it('shows seeds with the shared configuration list and an edit menu', async () => {
    const seeds = [
      new ConfigObject({
        id: 'seed-one',
        kind: Kind.SEED,
        meta: new Meta({
          name: 'First seed',
          description: 'Hidden seed description',
          labelList: [new Label({key: 'hidden', value: 'label'})],
        }),
      }),
      new ConfigObject({id: 'seed-two', kind: Kind.SEED, meta: new Meta({name: 'Second seed'})}),
    ];
    fixture.componentRef.setInput('seedDataSource', {
      rows$: of(seeds),
      reset$: EMPTY,
      completed$: EMPTY,
      appendLoading$: of(false),
      appendFailed$: of(false),
      snapshot: seeds,
      retry: vi.fn(),
      loadMore: vi.fn(),
    } as unknown as ListDataSource<ConfigObject, ConfigQuery>);
    fixture.detectChanges();
    await fixture.whenStable();

    const aside = fixture.nativeElement.querySelector('.seed-aside') as HTMLElement;
    const rows = [...aside.querySelectorAll('mat-list-item.item-row')] as HTMLElement[];
    const links = [...aside.querySelectorAll('a.primary-detail-link')] as HTMLAnchorElement[];
    expect(aside.querySelector('mat-card')).toBeNull();
    expect(aside.querySelector('mat-list')).not.toBeNull();
    expect(links.map(link => link.textContent.trim())).toEqual(['First seed', 'Second seed']);
    expect(rows[0].querySelector('[matListItemLine]')).toBeNull();
    expect(rows[0].querySelector('.label-region')).toBeNull();
    expect(rows[0].querySelector('.config-kind-icon')).toBeNull();
    expect(rows[0].textContent).not.toContain('Hidden seed description');
    expect(rows[0].textContent).not.toContain('hidden');
    expect(getComputedStyle(aside.querySelector('.seed-list mat-list')).paddingLeft).toBe('0px');
    expect(links.map(link => link.getAttribute('href'))).toEqual([
      '/config/seed/seed-one',
      '/config/seed/seed-two',
    ]);
    const moreActions = rows[0].querySelector('button[aria-label="More actions"]') as HTMLButtonElement;
    expect(moreActions).not.toBeNull();
    moreActions.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(document.body.textContent).toContain('Edit');
  });

  it('places label links above seeds in the same context aside', async () => {
    fixture.componentRef.setInput('labelLinks', [{
      text: 'Owner registry',
      href: 'https://example.com/owners/example',
    }]);
    fixture.componentRef.setInput('seedDataSource', {
      rows$: of([]), reset$: EMPTY, completed$: EMPTY,
      appendLoading$: of(false), appendFailed$: of(false), snapshot: [],
      retry: vi.fn(), loadMore: vi.fn(),
    } as unknown as ListDataSource<ConfigObject, ConfigQuery>);
    fixture.detectChanges();
    await fixture.whenStable();

    const aside = fixture.nativeElement.querySelector('.seed-aside') as HTMLElement;
    const links = aside.querySelector('app-config-label-links') as HTMLElement;
    const seeds = aside.querySelector('.seed-context') as HTMLElement;
    const contextLink = links.querySelector('a[mat-list-item]') as HTMLAnchorElement;
    expect(links).not.toBeNull();
    expect(seeds).not.toBeNull();
    expect(links.compareDocumentPosition(seeds) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(contextLink.querySelector('mat-icon')).toBeNull();
    expect(getComputedStyle(contextLink).cursor).toBe('pointer');
  });

  it('puts state filter chips on the Seeds title row and emits the selected state', async () => {
    fixture.componentRef.setInput('seedDataSource', {
      rows$: of([]), reset$: EMPTY, completed$: EMPTY,
      appendLoading$: of(false), appendFailed$: of(false), snapshot: [],
      retry: vi.fn(), loadMore: vi.fn(),
    } as unknown as ListDataSource<ConfigObject, ConfigQuery>);
    const changes: (boolean | null)[] = [];
    component.seedStatusChange.subscribe(value => changes.push(value));
    fixture.detectChanges();

    const heading = fixture.nativeElement.querySelector('.seed-list-heading') as HTMLElement;
    expect(heading.querySelector('h2').textContent.trim()).toBe('Seeds');
    expect(heading.querySelectorAll('mat-chip-option')).toHaveLength(2);
    const stateChips = await loader.getAllHarnesses(
      MatChipOptionHarness.with({ancestor: '.seed-list-heading'}),
    );
    await stateChips[1].select();

    expect(changes).toEqual([true]);
  });

  it('puts the create seed FAB before the Seeds title and emits the entity', () => {
    fixture.componentRef.setInput('seedDataSource', {
      rows$: of([]), reset$: EMPTY, completed$: EMPTY,
      appendLoading$: of(false), appendFailed$: of(false), snapshot: [],
      retry: vi.fn(), loadMore: vi.fn(),
    } as unknown as ListDataSource<ConfigObject, ConfigQuery>);
    const created: ConfigObject[] = [];
    component.createSeed.subscribe(entity => created.push(entity));
    fixture.detectChanges();

    const titleActions = fixture.nativeElement.querySelector('.seed-title-actions') as HTMLElement;
    const button = titleActions.querySelector('button.create-seed-fab') as HTMLButtonElement;
    const title = titleActions.querySelector('h2') as HTMLElement;
    expect(button).not.toBeNull();
    expect(button.compareDocumentPosition(title) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(button).boxShadow).toBe('none');

    button.click();
    expect(created).toEqual([component.configObject]);
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
