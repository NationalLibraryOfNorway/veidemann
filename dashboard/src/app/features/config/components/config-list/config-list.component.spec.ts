import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {Component, DestroyRef, ElementRef, ViewChild} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {MatMenuHarness} from '@angular/material/menu/testing';
import {MatSortHeaderHarness} from '@angular/material/sort/testing';
import {BehaviorSubject, of, Subject} from 'rxjs';
import {ConfigListComponent} from './config-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  ConfigRef,
  Kind,
  Label,
  ListDataSource,
  Meta,
  Role,
  RoleMapping,
  Seed,
} from '../../../../shared/models';
import {AppConfig} from '../../../../app.config';
import {PreviewDirective} from '../../../../shared/directives';

@Component({
  template: `
    <app-config-list [dataSource]="dataSource">
      <ng-template appPreview let-row><span class="preview-content">Preview {{ row.meta.name }}</span></ng-template>
    </app-config-list>
  `,
  imports: [ConfigListComponent, PreviewDirective],
  standalone: true,
})
class PreviewHostComponent {
  @ViewChild(ConfigListComponent, {static: true}) list: ConfigListComponent;
  dataSource: ListDataSource<ConfigObject, unknown>;
}

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;
  let loader: HarnessLoader;
  let appConfig: AppConfig;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        ConfigListComponent,
        PreviewHostComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ConfigListComponent);
    component = fixture.componentInstance;
    appConfig = TestBed.inject(AppConfig);
    appConfig.labelLinks = {};
    loader = TestbedHarnessEnvironment.loader(fixture);
    await fixture.whenStable();
  });

  afterEach(() => vi.unstubAllGlobals());

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders role mapping identities as titles and roles as subtitles', async () => {
    const emailMapping = new ConfigObject({
      id: 'email-mapping',
      kind: Kind.ROLEMAPPING,
      meta: new Meta({name: 'roleMapping'}),
      roleMapping: new RoleMapping({
        email: 'curator@example.test',
        roleList: [Role.CURATOR, Role.READONLY],
      }),
    });
    const groupMapping = new ConfigObject({
      id: 'group-mapping',
      kind: Kind.ROLEMAPPING,
      meta: new Meta({name: 'roleMapping'}),
      roleMapping: new RoleMapping({
        group: 'operators',
        roleList: [Role.OPERATOR, Role.SYSTEM],
      }),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(emailMapping, groupMapping),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const titles = fixture.nativeElement.querySelectorAll('.item-row [matlistitemtitle]') as NodeListOf<HTMLElement>;
    const subtitles = fixture.nativeElement.querySelectorAll('.item-row [matlistitemline]') as NodeListOf<HTMLElement>;
    const rowCheckboxes = await loader.getAllHarnesses(MatCheckboxHarness.with({selector: '.item-row .selection-control'}));
    expect(Array.from(titles, title => title.textContent.trim()))
      .toEqual(['curator@example.test', 'operators']);
    expect(Array.from(subtitles, subtitle => subtitle.textContent.trim()))
      .toEqual(['CURATOR, READONLY', 'OPERATOR, SYSTEM']);
    expect(await Promise.all(rowCheckboxes.map(checkbox => checkbox.getAriaLabel())))
      .toEqual(['Select curator@example.test', 'Select operators']);

    component.onCheckboxToggle(emailMapping);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.item-row [matlistitemtitle]').textContent.trim())
      .toBe('curator@example.test');
    expect(component.configTitle(new ConfigObject({
      kind: Kind.ROLEMAPPING,
      meta: new Meta({name: 'roleMapping fallback'}),
    }))).toBe('roleMapping fallback');
  });

  it('clears list and container selection when a new query resets the data source', () => {
    const query = new BehaviorSubject('first');
    const rows = new Subject<ConfigObject>();
    const selections: ConfigObject[][] = [];
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: () => rows,
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.dataSource = dataSource;
    component.selectedChange.subscribe(selected => selections.push(selected));

    rows.next(new ConfigObject({id: 'one'}));
    component.onMasterCheckboxToggle(true);
    expect(component.selectedRows()).toHaveLength(1);

    query.next('second');

    expect(component.selectedRows()).toEqual([]);
    expect(selections.at(-1)).toEqual([]);
  });

  it('shows row and master checkboxes from the start', async () => {
    const row = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const masterFromStart = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(await masterFromStart.isChecked()).toBe(false);
    expect(await masterFromStart.isIndeterminate()).toBe(false);
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
    const entry = await loader.getHarness(MatCheckboxHarness.with({selector: '.item-row .selection-control'}));
    expect(await entry.getAriaLabel()).toBe('Select One');
    expect(await entry.isChecked()).toBe(false);

    await entry.check();

    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(header.querySelector('.master-selection-control')).not.toBeNull();
    expect(await master.getAriaLabel()).toBe('Select all loaded configurations');
    expect(await master.isChecked()).toBe(true);
    expect(await master.isIndeterminate()).toBe(false);
    const masterControl = header.querySelector('.master-selection-control') as HTMLElement;
    const rowControl = fixture.nativeElement.querySelector('.item-row .selection-control') as HTMLElement;
    expect(Math.abs(horizontalCenter(masterControl) - horizontalCenter(rowControl))).toBeLessThanOrEqual(1);

    fixture.componentRef.setInput('multiSelect', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();
  });

  it('selects all shown rows from the header checkbox without a prior selection', async () => {
    const first = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', kind: Kind.SEED, meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(await master.isDisabled()).toBe(false);
    await master.check();
    expect(component.selection.selected).toEqual([first, second]);
    expect(fixture.nativeElement.querySelector('.selection-actions')).not.toBeNull();

    await master.uncheck();
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
  });

  it('uses checked and indeterminate master states and renders actions beside it', async () => {
    const first = new ConfigObject({id: 'one', meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.length = 2;
    component.onCheckboxToggle(first);
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(await master.isChecked()).toBe(false);
    expect(await master.isIndeterminate()).toBe(true);
    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const masterElement = header.querySelector('.master-selection-control') as HTMLElement;
    const actions = header.querySelector('.selection-actions') as HTMLElement;
    const summary = header.querySelector('.selection-summary') as HTMLElement;
    expect(masterElement.compareDocumentPosition(actions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(actions.compareDocumentPosition(summary) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(summary.textContent.replace(/\s+/g, ' ').trim()).toBe('1 configurations of 2 selected.');
    expect(getComputedStyle(summary).justifyContent).toBe('center');
    expect(getComputedStyle(summary).textAlign).toBe('center');
    expect(header.querySelector('.result-count')).toBeNull();

    await master.check();
    expect(component.selection.selected).toEqual([first, second]);
    expect(await master.isChecked()).toBe(true);
    expect(await master.isIndeterminate()).toBe(false);
    expect(summary.textContent.replace(/\s+/g, ' ').trim()).toBe('All 2 configurations of 2 selected.');

    await master.uncheck();
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
  });

  it('renders a centered, type-specific result count with row subtitle typography', () => {
    const first = new ConfigObject({
      id: 'one',
      kind: Kind.SEED,
      meta: new Meta({name: 'One', description: 'First seed'}),
    });
    const second = new ConfigObject({id: 'two', kind: Kind.SEED});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.configKind = Kind.SEED;
    component.length = 3842;
    fixture.detectChanges();

    const count = fixture.nativeElement.querySelector('.result-count') as HTMLElement;
    const subtitle = fixture.nativeElement.querySelector('.item-row .mat-mdc-list-item-line') as HTMLElement;
    expect(count.textContent.replace(/\s+/g, ' ').trim()).toBe('2 of 3,842 seeds');
    expect(getComputedStyle(count).justifyContent).toBe('center');
    expect(getComputedStyle(count).textAlign).toBe('center');
    expect(getComputedStyle(count).backgroundColor).toBe('rgba(0, 0, 0, 0)');
    expect(getComputedStyle(count).fontSize).toBe(getComputedStyle(subtitle).fontSize);
    expect(getComputedStyle(count).lineHeight).toBe(getComputedStyle(subtitle).lineHeight);
    expect(getComputedStyle(count).fontWeight).toBe('500');
    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    expect(Math.abs(horizontalCenter(count) - horizontalCenter(header))).toBeLessThanOrEqual(1);
  });

  it('renders evenly spaced state filters with standard icon-button appearance', async () => {
    const row = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.showStateFilter = true;
    component.showOrderControl = true;
    component.configKind = Kind.SEED;
    component.length = 0;
    const stateChanges: (boolean | null)[] = [];
    component.disabledFilterChange.subscribe(value => stateChanges.push(value));
    fixture.detectChanges();

    const headerControls = fixture.nativeElement.querySelector('.list-header-controls') as HTMLElement;
    const stateFilter = headerControls.querySelector('.state-filter') as HTMLElement;
    const orderControl = headerControls.querySelector('.order-control') as HTMLElement;
    const resultCount = fixture.nativeElement.querySelector('.result-count') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
    expect(orderControl.compareDocumentPosition(stateFilter) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(headerControls.compareDocumentPosition(resultCount) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    const orderIcon = orderControl.querySelector('mat-icon') as HTMLElement;

    const [deactivated, active] = [...stateFilter.querySelectorAll('button')] as HTMLButtonElement[];
    const [deactivatedHarness, activeHarness] = await loader.getAllHarnesses(MatButtonHarness.with({
      selector: '.state-filter .stateful-filter-button',
    }));
    expect(deactivated.querySelector('mat-icon')?.textContent.trim()).toBe('toggle_off');
    expect(deactivated.getAttribute('aria-label')).toBe('Deactivated');
    expect(deactivated.getAttribute('mattooltip')).toBe('Deactivated');
    expect(deactivated.getAttribute('aria-pressed')).toBe('false');
    expect(active.querySelector('mat-icon')?.textContent.trim()).toBe('toggle_on');
    expect(active.getAttribute('aria-label')).toBe('Active');
    expect(active.getAttribute('mattooltip')).toBe('Active');
    expect(active.getAttribute('aria-pressed')).toBe('false');
    expect(deactivated.classList).toContain('stateful-filter-button');
    expect(active.classList).toContain('stateful-filter-button');
    expect(deactivated.classList).toContain('mat-mdc-icon-button');
    expect(active.classList).toContain('mat-mdc-icon-button');
    expect(await deactivatedHarness.getVariant()).toBe('icon');
    expect(await activeHarness.getVariant()).toBe('icon');
    expect(active.querySelector('.mat-mdc-button-persistent-ripple')).not.toBeNull();
    expect(getComputedStyle(deactivated).color).toBe(getComputedStyle(orderControl).color);
    expect(getComputedStyle(active).color).toBe(getComputedStyle(orderControl).color);
    expect(getComputedStyle(deactivated.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
    expect(getComputedStyle(active.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
    const orderToDeactivated = Math.abs(horizontalCenter(orderIcon) - horizontalCenter(
      deactivated.querySelector('mat-icon') as HTMLElement
    ));
    const deactivatedToActive = Math.abs(horizontalCenter(
      deactivated.querySelector('mat-icon') as HTMLElement
    ) - horizontalCenter(active.querySelector('mat-icon') as HTMLElement));
    expect(orderToDeactivated).toBeCloseTo(deactivatedToActive, 1);
    const inactiveColor = getComputedStyle(active).color;

    active.click();
    expect(stateChanges.at(-1)).toBe(false);
    fixture.componentRef.setInput('disabledFilter', false);
    fixture.detectChanges();
    expect(active.getAttribute('aria-pressed')).toBe('true');
    expect(deactivated.getAttribute('aria-pressed')).toBe('false');
    expect(getComputedStyle(active.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 1');
    expect(getComputedStyle(active).color).not.toBe(inactiveColor);

    deactivated.click();
    expect(stateChanges.at(-1)).toBe(true);
    fixture.componentRef.setInput('disabledFilter', true);
    fixture.detectChanges();
    expect(active.getAttribute('aria-pressed')).toBe('false');
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');
    expect(getComputedStyle(active.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
    expect(getComputedStyle(deactivated.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 1');

    deactivated.click();
    expect(stateChanges.at(-1)).toBeNull();
    fixture.componentRef.setInput('disabledFilter', null);
    fixture.detectChanges();
    expect(deactivated.getAttribute('aria-pressed')).toBe('false');
    expect(getComputedStyle(deactivated).color).toBe(inactiveColor);
    expect(getComputedStyle(deactivated.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
  });

  it('offers every supported order and emits the corresponding sort', async () => {
    component.showOrderControl = true;
    component.length = 0;
    const sorts: {active: string; direction: string}[] = [];
    component.sort.subscribe(value => sorts.push(value));
    fixture.detectChanges();

    const button = await loader.getHarness(MatButtonHarness.with({
      selector: '[data-testid="configuration-order"]',
      variant: 'icon',
      iconName: 'sort',
    }));
    expect(await button.getVariant()).toBe('icon');
    const orderButton = fixture.nativeElement.querySelector('.order-control') as HTMLButtonElement;
    expect(orderButton.tagName).toBe('BUTTON');
    expect(orderButton.classList).toContain('order-control-icon-only');
    expect(orderButton.classList).toContain('mat-mdc-icon-button');
    expect(orderButton.querySelector('.mat-mdc-button-persistent-ripple')).not.toBeNull();
    expect(orderButton.querySelector('.order-label')).toBeNull();
    expect(orderButton.querySelector('.order-menu-indicator')).toBeNull();
    expect(fixture.nativeElement.querySelector('.list-header-controls mat-form-field')).toBeNull();
    expect(fixture.nativeElement.querySelector('.list-header-controls mat-select')).toBeNull();

    let menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    const options = await menu.getItems();
    expect(await Promise.all(options.map(option => option.getText()))).toEqual([
      'Default order',
      'Name: A–Z',
      'Name: Z–A',
      'Last modified: newest first',
      'Last modified: oldest first',
    ]);
    expect(await options[0].isDisabled()).toBe(true);

    await options[1].click();
    expect(sorts.at(-1)).toEqual({active: 'name', direction: 'asc'});
    fixture.componentRef.setInput('sortActive', 'name');
    fixture.componentRef.setInput('sortDirection', 'asc');
    fixture.detectChanges();
    const selectedOrderButton = fixture.nativeElement.querySelector('.order-control') as HTMLButtonElement;
    expect(selectedOrderButton.classList).not.toContain('order-control-icon-only');
    expect(selectedOrderButton.classList).not.toContain('mat-mdc-icon-button');
    const orderLabel = selectedOrderButton.querySelector('.order-label') as HTMLElement;
    const orderIndicator = selectedOrderButton.querySelector('.order-menu-indicator') as HTMLElement;
    expect(orderLabel.textContent?.trim()).toBe('Name: A–Z');
    expect(orderLabel.compareDocumentPosition(orderIndicator) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    await menu.clickItem({text: 'Name: Z–A'});
    expect(sorts.at(-1)).toEqual({active: 'name', direction: 'desc'});
    fixture.componentRef.setInput('sortDirection', 'desc');
    fixture.detectChanges();

    menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    await menu.clickItem({text: 'Last modified: newest first'});
    expect(sorts.at(-1)).toEqual({active: 'lastModified', direction: 'desc'});
    fixture.componentRef.setInput('sortActive', 'lastModified');
    fixture.detectChanges();

    menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    await menu.clickItem({text: 'Last modified: oldest first'});
    expect(sorts.at(-1)).toEqual({active: 'lastModified', direction: 'asc'});
    fixture.componentRef.setInput('sortDirection', 'asc');
    fixture.detectChanges();

    menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    await menu.clickItem({text: 'Default order'});
    expect(sorts.at(-1)).toEqual({active: '', direction: ''});
  });

  it('sorts by name via the sort-header arrow next to the order control, in sync with it', async () => {
    component.showOrderControl = true;
    component.length = 0;
    const sorts: {active: string; direction: string}[] = [];
    component.sort.subscribe(value => sorts.push(value));
    fixture.detectChanges();

    const nameHeader = await loader.getHarness(MatSortHeaderHarness.with({label: 'Name'}));
    expect(await nameHeader.isDisabled()).toBe(false);
    expect(await nameHeader.isActive()).toBe(false);

    await nameHeader.click();
    expect(sorts.at(-1)).toEqual({active: 'name', direction: 'asc'});
    fixture.componentRef.setInput('sortActive', 'name');
    fixture.componentRef.setInput('sortDirection', 'asc');
    fixture.detectChanges();
    expect(await nameHeader.isActive()).toBe(true);
    expect(await nameHeader.getSortDirection()).toBe('asc');
    expect(fixture.nativeElement.querySelector('.order-control')?.textContent).toContain('Name: A–Z');

    await nameHeader.click();
    expect(sorts.at(-1)).toEqual({active: 'name', direction: 'desc'});
    fixture.componentRef.setInput('sortDirection', 'desc');
    fixture.detectChanges();
    expect(await nameHeader.getSortDirection()).toBe('desc');
    expect(fixture.nativeElement.querySelector('.order-control')?.textContent).toContain('Name: Z–A');

    await nameHeader.click();
    expect(sorts.at(-1)).toEqual({active: 'name', direction: ''});

    component.selectedRows.set([new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})})]);
    fixture.detectChanges();
    expect(component.isSelectionMode()).toBe(true);
    expect(await nameHeader.isDisabled()).toBe(true);
  });

  it('prepends selection controls while preserving filtering and ordering', async () => {
    const row = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.showStateFilter = true;
    component.showOrderControl = true;
    component.disabledFilter = true;
    component.sortActive = 'name';
    component.sortDirection = 'asc';
    component.length = 1;
    component.onCheckboxToggle(row);
    const sorts: {active: string; direction: string}[] = [];
    const stateChanges: (boolean | null)[] = [];
    component.sort.subscribe(value => sorts.push(value));
    component.disabledFilterChange.subscribe(value => stateChanges.push(value));
    fixture.detectChanges();

    expect(component.isSelectionMode()).toBe(true);
    const leading = fixture.nativeElement.querySelector('.list-header-leading') as HTMLElement;
    const selection = leading.querySelector('.selection-leading') as HTMLElement;
    const master = selection.querySelector('.master-selection-control') as HTMLElement;
    const bulk = selection.querySelector('.selection-actions') as HTMLElement;
    const controls = leading.querySelector('.list-header-controls') as HTMLElement;
    const order = controls.querySelector('.order-control') as HTMLElement;
    const states = controls.querySelector('.state-filter') as HTMLElement;

    expect(master.compareDocumentPosition(bulk) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(selection.compareDocumentPosition(controls) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(order.compareDocumentPosition(states) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(controls).marginInlineStart).toBe('24px');
    expect(order.textContent).toContain('Name: A–Z');
    expect((order as HTMLButtonElement).disabled).toBe(true);

    const [deactivated, active] = [...states.querySelectorAll('button')] as HTMLButtonElement[];
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');
    expect(deactivated.disabled).toBe(true);
    expect(active.disabled).toBe(true);
    component.onOrderChange('lastModified:desc');
    component.toggleDisabledFilter(false);
    expect(sorts).toEqual([]);
    expect(stateChanges).toEqual([]);

    component.onDeselectAll();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
    expect(fixture.nativeElement.querySelector('.order-control')?.textContent).toContain('Name: A–Z');
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');
    expect((fixture.nativeElement.querySelector('.order-control') as HTMLButtonElement).disabled).toBe(false);
    expect((fixture.nativeElement.querySelector('.stateful-filter-button') as HTMLButtonElement).disabled).toBe(false);
  });

  it('automatically selects appended rows while loaded-row selection is active', async () => {
    const rows = new Subject<ConfigObject>();
    const first = new ConfigObject({id: 'one', kind: component.Kind.SEED, meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', kind: component.Kind.SEED, meta: new Meta({name: 'Two'})});
    const third = new ConfigObject({id: 'three', kind: component.Kind.SEED, meta: new Meta({name: 'Three'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => rows,
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const selections: ConfigObject[][] = [];
    component.selectedChange.subscribe(value => selections.push([...value]));
    rows.next(first);
    rows.next(second);
    fixture.detectChanges();
    component.onCheckboxToggle(first);
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));

    await master.check();
    rows.next(third);
    fixture.detectChanges();

    expect(component.selection.selected).toEqual([first, second, third]);
    expect(selections.at(-1)).toEqual([first, second, third]);
    expect(await master.isChecked()).toBe(true);
  });

  it('offers database selection only through a chip and changes it to removal', async () => {
    const rows = new Subject<ConfigObject>();
    const first = new ConfigObject({id: 'one', kind: component.Kind.SEED, meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', kind: component.Kind.SEED, meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => rows,
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.length = 3;
    const selections: ConfigObject[][] = [];
    let selectAllCount = 0;
    component.selectedChange.subscribe(value => selections.push([...value]));
    component.selectAll.subscribe(() => selectAllCount++);
    rows.next(first);
    fixture.detectChanges();
    component.onCheckboxToggle(first);
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));

    let chip = fixture.nativeElement.querySelector('.database-selection-chip') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.selection-summary').textContent.replace(/\s+/g, ' ').trim())
      .toContain('All 1 seeds of 3 selected.');
    expect(chip.textContent.trim()).toBe('Select all 3 seeds in the database.');
    chip.click();
    fixture.detectChanges();

    expect(selectAllCount).toBe(1);
    expect(component.allSelected()).toBe(true);
    expect(await master.isChecked()).toBe(true);
    expect(fixture.nativeElement.querySelector('.selection-summary').textContent)
      .toContain('All 3 seeds in the database are selected.');
    chip = fixture.nativeElement.querySelector('.database-selection-chip') as HTMLElement;
    expect(chip.textContent.trim()).toBe('Remove the selection');

    const emissionCount = selections.length;
    rows.next(second);
    fixture.detectChanges();
    expect(component.allSelected()).toBe(true);
    expect(component.selection.selected).toEqual([first, second]);
    expect(selections).toHaveLength(emissionCount);

    await master.uncheck();
    fixture.detectChanges();
    expect(component.allSelected()).toBe(false);
    expect(component.selection.selected).toEqual([]);

    component.onCheckboxToggle(first);
    component.onMasterCheckboxToggle(true);
    fixture.detectChanges();
    chip = fixture.nativeElement.querySelector('.database-selection-chip') as HTMLElement;
    chip.click();
    fixture.detectChanges();
    chip = fixture.nativeElement.querySelector('.database-selection-chip') as HTMLElement;
    const space = new KeyboardEvent('keydown', {key: ' ', bubbles: true, cancelable: true});
    chip.dispatchEvent(space);
    fixture.detectChanges();
    expect(space.defaultPrevented).toBe(true);
    expect(component.allSelected()).toBe(false);
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-summary')).toBeNull();
  });

  it('uses type-specific selection summaries while the total is unavailable', async () => {
    const first = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({description: 'First'})});
    const second = new ConfigObject({id: 'two', kind: Kind.SEED, meta: new Meta({description: 'Second'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.configKind = Kind.SEED;
    component.onCheckboxToggle(first);
    fixture.detectChanges();

    const summary = fixture.nativeElement.querySelector('.selection-summary') as HTMLElement;
    const subtitle = fixture.nativeElement.querySelector('.item-row .mat-mdc-list-item-line') as HTMLElement;
    expect(summary.textContent.replace(/\s+/g, ' ').trim()).toBe('1 seeds selected.');
    expect(getComputedStyle(summary).fontSize).toBe(getComputedStyle(subtitle).fontSize);
    expect(getComputedStyle(summary).lineHeight).toBe(getComputedStyle(subtitle).lineHeight);

    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    await master.check();
    expect(summary.textContent.replace(/\s+/g, ' ').trim()).toBe('All 2 seeds selected.');
  });

  it('does not offer database selection when all matching rows are loaded', () => {
    const row = new ConfigObject({id: 'one', meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.length = 1;
    fixture.detectChanges();
    component.onCheckboxToggle(row);
    fixture.detectChanges();
    expect(component.allSelected()).toBe(false);
    expect(fixture.nativeElement.querySelector('.database-selection-chip')).toBeNull();
  });

  it.each([
    [Kind.CRAWLENTITY, 'entities'],
    [Kind.SEED, 'seeds'],
    [Kind.CRAWLJOB, 'crawl jobs'],
    [Kind.CRAWLSCHEDULECONFIG, 'schedules'],
    [Kind.CRAWLCONFIG, 'crawl configurations'],
    [Kind.BROWSERCONFIG, 'browser configurations'],
    [Kind.BROWSERSCRIPT, 'browser scripts'],
    [Kind.POLITENESSCONFIG, 'politeness configurations'],
    [Kind.CRAWLHOSTGROUPCONFIG, 'crawl host groups'],
    [Kind.ROLEMAPPING, 'role mappings'],
  ])('uses the route-specific plural label for kind %s', (kind, label) => {
    component.configKind = kind;

    expect(component.configTypePluralLabel()).toBe(label);
  });

  it('prefers the supplied route kind over the selected row kind and falls back to the row kind', () => {
    component.selectedRows.set([new ConfigObject({id: 'selected', kind: Kind.CRAWLJOB})]);
    expect(component.configTypePluralLabel()).toBe('crawl jobs');

    component.configKind = Kind.SEED;
    expect(component.configTypePluralLabel()).toBe('seeds');
  });

  it('only treats disabled Seeds and Crawl Jobs as deactivated configurations', () => {
    const seed = new ConfigObject({kind: Kind.SEED});
    seed.seed = {disabled: true} as ConfigObject['seed'];
    const crawlJob = new ConfigObject({kind: Kind.CRAWLJOB});
    crawlJob.crawlJob = {disabled: true} as ConfigObject['crawlJob'];
    const collection = new ConfigObject({kind: Kind.COLLECTION});
    collection.crawlJob = {disabled: true} as ConfigObject['crawlJob'];

    expect(component.isDeactivated(seed)).toBe(true);
    expect(component.isDeactivated(crawlJob)).toBe(true);
    expect(component.isDeactivated(collection)).toBe(false);
  });

  it('applies the selected state to the entire list item', () => {
    const row = new ConfigObject({id: 'one'});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    component.onCheckboxToggle(row);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.item-row').classList).toContain('row-checked');
  });

  it('centers title and supporting text against the leading checkbox without an offset', () => {
    const row = new ConfigObject({
      id: 'aligned',
      meta: new Meta({name: 'Aligned', description: 'Supporting text'}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const text = fixture.nativeElement.querySelectorAll('.item-row .row-text') as NodeListOf<HTMLElement>;
    expect(text).toHaveLength(2);
    expect(Array.from(text).some(element => getComputedStyle(element).top === '-4px')).toBe(false);
  });

  it('renders a supporting text line only when the configuration has a description', () => {
    const described = new ConfigObject({id: 'one', meta: new Meta({name: 'One', description: 'Supporting text'})});
    const plain = new ConfigObject({id: 'two', meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(described, plain),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    expect(rows[0].querySelector('.mat-mdc-list-item-line')).not.toBeNull();
    expect(rows[1].querySelector('.mat-mdc-list-item-line')).toBeNull();
  });

  it('renders labels in an accessible trailing column before row actions', () => {
    const labeled = new ConfigObject({
      id: 'labeled',
      meta: new Meta({
        name: 'Labeled configuration',
        labelList: [
          new Label({key: 'owner', value: 'archive'}),
          new Label({key: 'profile', value: 'default'}),
        ],
      }),
    });
    const unlabeled = new ConfigObject({
      id: 'unlabeled',
      meta: new Meta({name: 'Unlabeled configuration'}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(labeled, unlabeled),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });

    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    const labelRegion = rows[0].querySelector('.label-region') as HTMLElement;
    const actionRegion = rows[0].querySelector('.action-region') as HTMLElement;
    const chips = labelRegion.querySelectorAll('mat-chip') as NodeListOf<HTMLElement>;
    expect(rows[0].classList).toContain('row-with-labels');
    expect(rows[1].classList).not.toContain('row-with-labels');
    expect(labelRegion.parentElement).toBe(rows[0]);
    expect(actionRegion.parentElement).toBe(rows[0]);
    expect(Array.from(rows[0].children).indexOf(labelRegion))
      .toBeLessThan(Array.from(rows[0].children).indexOf(actionRegion));
    expect(labelRegion.hasAttribute('matlistitemmeta')).toBe(true);
    expect(labelRegion.hasAttribute('matlistitemline')).toBe(false);
    expect(chips).toHaveLength(2);
    expect(chips[0].textContent.trim()).toBe('owner:archive');
    expect(chips[0].getAttribute('aria-label')).toBe('Search for exact label owner:archive');
    expect(chips[0].classList).not.toContain('emoji-label-chip');
    expect(chips[1].textContent.trim()).toBe('profile:default');
    expect(chips[1].classList).not.toContain('emoji-label-chip');
    expect(rows[1].querySelector('.label-region')).toBeNull();
    expect(labelRegion.querySelector('mat-chip-set').getAttribute('aria-label')).toBe('Configuration labels');
  });

  it('renders only valid Unicode emoji labels as borderless glyph chips', () => {
    const emoji = new Label({key: 'emoji', value: '🐶'});
    const invalidEmoji = new Label({key: 'emoji', value: 'dog'});
    const row = new ConfigObject({
      id: 'emoji',
      meta: new Meta({name: 'Emoji configuration', labelList: [emoji, invalidEmoji]}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const chips = fixture.nativeElement.querySelectorAll('mat-chip') as NodeListOf<HTMLElement>;
    expect(chips[0].classList).toContain('emoji-label-chip');
    expect(chips[0].querySelector('.label-display__emoji').textContent).toBe('🐶');
    expect(chips[0].querySelector('.label-display__accessible').textContent).toBe('emoji:🐶');
    expect(chips[0].getAttribute('aria-label')).toBe('Search for exact label emoji:🐶');
    expect(chips[1].classList).not.toContain('emoji-label-chip');
    expect(chips[1].textContent.trim()).toBe('emoji:dog');
    expect(chips[1].querySelector('.label-display__emoji')).toBeNull();
  });

  it('activates a label once with mouse, Enter, or Space without activating the row', () => {
    const label = new Label({key: 'owner', value: 'archive'});
    const row = new ConfigObject({
      id: 'labeled',
      meta: new Meta({name: 'Labeled configuration', labelList: [label]}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const labels: Label[] = [];
    const rows: ConfigObject[] = [];
    const selections: ConfigObject[][] = [];
    component.labelClick.subscribe(value => labels.push(value));
    component.rowClick.subscribe(value => rows.push(value));
    component.selectedChange.subscribe(value => selections.push(value));
    fixture.detectChanges();

    const chip = fixture.nativeElement.querySelector('mat-chip') as HTMLElement;
    chip.click();
    expect(labels).toEqual([label]);

    const enter = new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true});
    chip.dispatchEvent(enter);
    expect(labels).toEqual([label, label]);
    expect(enter.defaultPrevented).toBe(true);

    const space = new KeyboardEvent('keydown', {key: ' ', bubbles: true, cancelable: true});
    chip.dispatchEvent(space);
    expect(labels).toEqual([label, label, label]);
    expect(space.defaultPrevented).toBe(true);
    expect(rows).toEqual([]);
    expect(selections).toEqual([]);
  });

  it('does not add configured external-link chips to list rows', () => {
    appConfig.labelLinks = {
      organisasjonsnummer: {
        text: 'Brønnøysundregistrene',
        urlTemplate: 'https://virksomhet.brreg.no/nb/oppslag/enheter/{value}',
      },
    };
    const label = new Label({key: 'organisasjonsnummer', value: '976 029/100'});
    const row = new ConfigObject({
      id: 'labeled',
      meta: new Meta({name: 'Labeled configuration', labelList: [label]}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const selectedLabels: Label[] = [];
    component.labelClick.subscribe(value => selectedLabels.push(value));
    fixture.detectChanges();

    const searchChip = fixture.nativeElement.querySelector('mat-chip') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.external-label-link')).toBeNull();
    expect(fixture.nativeElement.querySelectorAll('.label-region mat-chip')).toHaveLength(1);
    searchChip.click();
    expect(selectedLabels).toEqual([label]);
  });

  it('makes names and labels passive and hides nested actions in selection mode', () => {
    appConfig.labelLinks = {
      owner: {
        text: 'Owner registry',
        urlTemplate: 'https://example.com/{value}',
      },
    };
    const row = new ConfigObject({
      id: 'labeled',
      kind: Kind.SEED,
      meta: new Meta({name: 'Labeled', labelList: [new Label({key: 'owner', value: 'archive'})]}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.primary-detail-link')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.external-label-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.action-region')).not.toBeNull();

    (fixture.nativeElement.querySelector('.item-row .selection-control input') as HTMLElement).click();
    fixture.detectChanges();

    const chip = fixture.nativeElement.querySelector('.label-region mat-chip') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.primary-detail-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.external-label-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.action-region')).toBeNull();
    expect(chip.getAttribute('role')).not.toBe('button');
    expect(chip.hasAttribute('tabindex')).toBe(false);
  });

  it('only selects rows through the checkbox in selection mode and Escape clears selection', () => {
    const first = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', kind: Kind.SEED, meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const navigated: ConfigObject[] = [];
    component.rowClick.subscribe(value => navigated.push(value));
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.item-row .selection-control input') as HTMLElement).click();
    fixture.detectChanges();
    const rows = fixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    rows[1].click();
    rows[1].dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true}));
    expect(component.selection.selected).toEqual([first]);
    expect(navigated).toEqual([]);

    (rows[1].querySelector('.selection-control input') as HTMLElement).click();
    expect(component.selection.selected).toEqual([first, second]);

    const escape = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
    rows[0].dispatchEvent(escape);
    fixture.detectChanges();
    expect(escape.defaultPrevented).toBe(true);
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
  });

  it('preserves row navigation, deactivated styling, and bulk selection behavior', () => {
    const enabled = new ConfigObject({
      id: 'enabled',
      meta: new Meta({name: 'Enabled'}),
    });
    const disabled = new ConfigObject({
      id: 'disabled',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Disabled'}),
    });
    disabled.crawlJob = {disabled: true} as ConfigObject['crawlJob'];
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(enabled, disabled),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const rowClicks: ConfigObject[] = [];
    const selections: ConfigObject[][] = [];
    let selectAllCount = 0;
    component.rowClick.subscribe(value => rowClicks.push(value));
    component.selectedChange.subscribe(value => selections.push(value));
    component.selectAll.subscribe(() => selectAllCount++);
    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    expect(rows[0].classList).not.toContain('row-deactivated');
    expect(rows[1].classList).toContain('row-deactivated');
    expect(rows[0].hasAttribute('aria-description')).toBe(false);
    expect(rows[1].getAttribute('aria-description')).toBe('Deactivated');
    expect(rows[1].querySelector('.deactivated-status')).toBeNull();
    expect(rows[1].querySelector('.action-region')).not.toBeNull();

    rows[0].click();
    rows[0].dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    (rows[0].querySelector('.primary-detail-link') as HTMLElement).click();
    rows[1].click();
    expect(rowClicks).toEqual([enabled, enabled, enabled, disabled]);

    component.onMasterCheckboxToggle(true);
    fixture.detectChanges();
    expect(selections.at(-1)).toEqual([enabled, disabled]);
    expect(component.isAllLoadedSelected()).toBe(true);
    expect(rows[1].classList).toContain('row-checked');
    expect(rows[1].getAttribute('aria-description')).toBe('Deactivated');
    component.onSelectAll();
    expect(selectAllCount).toBe(1);
    expect(component.allSelected()).toBe(true);
    component.onDeselectAll();
    expect(component.selection.selected).toEqual([]);
    expect(component.allSelected()).toBe(false);
  });

  it('keeps Collections non-selectable', () => {
    const collection = new ConfigObject({
      id: 'collection',
      kind: Kind.COLLECTION,
      meta: new Meta({name: 'Collection'}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(collection),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('multiSelect', false);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.selection-control')).toBeNull();
    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-detail-link')).not.toBeNull();
  });

  it('retries failed loads and requests another page when the sentinel intersects', () => {
    let intersectionCallback!: IntersectionObserverCallback;
    const observe = vi.fn();
    const disconnect = vi.fn();
    class TestIntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback;
      }

      observe = observe;
      disconnect = disconnect;
      unobserve = vi.fn();
      takeRecords = vi.fn(() => []);
      root = null;
      rootMargin = '';
      thresholds = [];
    }
    vi.stubGlobal('IntersectionObserver', TestIntersectionObserver);
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(new ConfigObject({id: 'one'})),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const retry = vi.spyOn(dataSource, 'retry');
    const loadMore = vi.spyOn(dataSource, 'loadMore');
    component.dataSource = dataSource;

    component.retry();
    const sentinel = document.createElement('div');
    component.loadMoreSentinel = new ElementRef(sentinel);
    intersectionCallback([{isIntersecting: true} as IntersectionObserverEntry], {} as IntersectionObserver);

    expect(retry).toHaveBeenCalledTimes(1);
    expect(observe).toHaveBeenCalledWith(sentinel);
    expect(loadMore).toHaveBeenCalledTimes(1);
  });
  it('combines seed row decorations for labels and crawl jobs', async () => {
    const seed = new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      meta: new Meta({name: 'https://example.test/', labelList: [new Label({key: 'topic', value: 'news'})]}),
      seed: new Seed({jobRefList: [
        new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}),
        new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-unknown'}),
      ]}),
    });
    component.configKind = Kind.SEED;
    component.showDecorationToggle = true;
    component.crawlJobs = [new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily crawl'})})];
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(seed),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const crawlJobClicks: string[] = [];
    const rowClicks: ConfigObject[] = [];
    component.crawlJobClick.subscribe(id => crawlJobClicks.push(id));
    component.rowClick.subscribe(row => rowClicks.push(row));
    fixture.detectChanges();

    const chipTexts = () => ([...fixture.nativeElement.querySelectorAll('.item-row .label-region mat-chip')] as HTMLElement[])
      .map(chip => chip.textContent.trim());
    const toggleGroup = fixture.nativeElement.querySelector('[data-testid="configuration-row-decoration"]') as HTMLElement;
    expect(toggleGroup.closest('.header-trailing')).not.toBeNull();
    const toggles = [...toggleGroup.querySelectorAll('button')] as HTMLButtonElement[];
    const toggle = (index: number) => {
      toggles[index].click();
      fixture.detectChanges();
    };
    expect(toggles.length).toBe(2);
    expect(toggles[0].getAttribute('aria-pressed')).toBe('true');
    expect(toggles[1].getAttribute('aria-pressed')).toBe('false');
    expect(chipTexts()).toEqual([expect.stringContaining('topic')]);

    toggle(1);
    expect(toggles[0].getAttribute('aria-pressed')).toBe('true');
    expect(toggles[1].getAttribute('aria-pressed')).toBe('true');
    expect(chipTexts()).toEqual([expect.stringContaining('topic'), expect.stringContaining('Daily crawl'),
      expect.stringContaining('job-unknown')]);
    expect(fixture.nativeElement.querySelector('.label-region mat-chip-set').getAttribute('aria-label'))
      .toBe('Configuration labels and crawl jobs');

    toggle(0);
    expect(toggles[0].getAttribute('aria-pressed')).toBe('false');
    expect(chipTexts().map(text => text.replace(/^\S+\s+/, ''))).toEqual(['Daily crawl', 'job-unknown']);
    const jobChip = fixture.nativeElement.querySelector('.item-row .crawl-job-chip') as HTMLElement;
    expect(jobChip.getAttribute('aria-label')).toBe('Filter by crawl job Daily crawl');
    jobChip.click();
    jobChip.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(crawlJobClicks).toEqual(['job-1', 'job-1']);
    expect(rowClicks).toEqual([]);

    toggle(1);
    expect(fixture.nativeElement.querySelector('.item-row .label-region')).toBeNull();
    expect(fixture.nativeElement.querySelector('.item-row').classList).not.toContain('row-with-labels');
  });

  it('hides the decoration toggle unless enabled', () => {
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.showOrderControl = true;
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[data-testid="configuration-row-decoration"]')).toBeNull();
  });

  describe('expandable rows', () => {
    let hostFixture: ComponentFixture<PreviewHostComponent>;
    let first: ConfigObject;
    let second: ConfigObject;
    let rowClicks: ConfigObject[];

    const rows = () => hostFixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    const previews = () => hostFixture.nativeElement.querySelectorAll('.row-preview') as NodeListOf<HTMLElement>;

    beforeEach(() => {
      first = new ConfigObject({id: 'first', kind: Kind.COLLECTION, meta: new Meta({name: 'First'})});
      second = new ConfigObject({id: 'second', kind: Kind.COLLECTION, meta: new Meta({name: 'Second'})});
      hostFixture = TestBed.createComponent(PreviewHostComponent);
      hostFixture.componentInstance.dataSource = ListDataSource.fromQuery({
        query$: of('query'),
        load: () => of(first, second),
        destroyRef: hostFixture.componentRef.injector.get(DestroyRef),
      });
      hostFixture.detectChanges();
      component = hostFixture.componentInstance.list;
      rowClicks = [];
      component.rowClick.subscribe(value => rowClicks.push(value));
    });

    it('toggles a preview below the row on click and Enter instead of emitting rowClick', () => {
      expect(previews().length).toBe(0);
      expect(rows()[0].getAttribute('aria-expanded')).toBe('false');

      rows()[0].click();
      hostFixture.detectChanges();
      expect(previews().length).toBe(1);
      expect(rows()[0].nextElementSibling).toBe(previews()[0]);
      expect(previews()[0].querySelector('.preview-content')?.textContent).toBe('Preview First');
      expect(previews()[0].getAttribute('role')).toBe('listitem');
      expect(rows()[0].getAttribute('aria-expanded')).toBe('true');
      expect(rows()[0].getAttribute('aria-controls')).toBe(previews()[0].id);

      rows()[0].click();
      hostFixture.detectChanges();
      expect(previews().length).toBe(0);

      rows()[0].dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
      hostFixture.detectChanges();
      expect(previews().length).toBe(1);
      expect(rowClicks).toEqual([]);
    });

    it('limits the title link to its text so the rest of the row toggles the preview', () => {
      const link = rows()[0].querySelector('.primary-detail-link') as HTMLElement;
      expect(getComputedStyle(link).width).toBe('fit-content');
      expect(getComputedStyle(link).maxWidth).toBe('100%');
    });

    it('keeps only one preview open at a time', () => {
      rows()[0].click();
      hostFixture.detectChanges();
      rows()[1].click();
      hostFixture.detectChanges();

      expect(previews().length).toBe(1);
      expect(rows()[1].nextElementSibling).toBe(previews()[0]);
      expect(rows()[0].getAttribute('aria-expanded')).toBe('false');
    });

    it('navigates through the title link and toggles through the expand button', () => {
      (rows()[0].querySelector('.primary-detail-link') as HTMLElement).click();
      hostFixture.detectChanges();
      expect(rowClicks).toEqual([first]);
      expect(previews().length).toBe(0);

      const toggle = rows()[1].querySelector('.preview-toggle') as HTMLElement;
      expect(toggle.getAttribute('aria-expanded')).toBe('false');
      toggle.click();
      hostFixture.detectChanges();
      expect(previews().length).toBe(1);
      expect(toggle.getAttribute('aria-expanded')).toBe('true');
      expect(toggle.getAttribute('aria-label')).toBe('Hide preview of Second');
      expect(rowClicks).toEqual([first]);
    });

    it('toggles the preview on row click in selection mode without changing the selection', () => {
      component.onCheckboxToggle(second);
      hostFixture.detectChanges();

      rows()[0].click();
      hostFixture.detectChanges();
      expect(previews().length).toBe(1);
      expect(rows()[0].nextElementSibling).toBe(previews()[0]);
      expect(rows()[0].getAttribute('aria-expanded')).toBe('true');
      expect(component.selection.selected.map(row => row.id)).toEqual(['second']);

      rows()[0].click();
      hostFixture.detectChanges();
      expect(previews().length).toBe(0);
    });

    it('keeps an open preview when selection mode starts', () => {
      rows()[0].click();
      hostFixture.detectChanges();
      component.onCheckboxToggle(second);
      hostFixture.detectChanges();

      expect(previews().length).toBe(1);
      expect(rows()[0].querySelector('.action-region')).toBeNull();
    });
  });

  describe('with fixed pages', () => {
    const all = ['a', 'b', 'c', 'd'].map(id => new ConfigObject({id, kind: Kind.SEED, meta: new Meta({name: id})}));
    let dataSource: ListDataSource<ConfigObject, string>;

    const ids = (rows: readonly ConfigObject[]) => rows.map(row => row.id);

    beforeEach(() => {
      dataSource = ListDataSource.fromQuery({
        query$: of('query'),
        load: (_query, range) => of(...all.slice(range.offset, range.offset + range.pageSize)),
        destroyRef: fixture.componentRef.injector.get(DestroyRef),
      });
      component.dataSource = dataSource;
      component.length = all.length;
      fixture.detectChanges();
      dataSource.setPaging(2);
      fixture.detectChanges();
    });

    it('keeps rows selected on earlier pages and counts them in the selection', () => {
      component.onCheckboxToggle(all[0]);
      dataSource.goToPage(1);
      fixture.detectChanges();

      expect(ids(dataSource.snapshot)).toEqual(['c', 'd']);
      expect(ids(component.selectedRows())).toEqual(['a']);
      expect(component.isSelectionMode()).toBe(true);
      expect(component.isAllLoadedSelected()).toBe(false);

      component.onCheckboxToggle(all[2]);
      expect(ids(component.selectedRows())).toEqual(['a', 'c']);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.selection-summary').textContent)
        .toContain('2 seeds of 4 selected');
    });

    it('lets the master checkbox select and deselect only the rows on the page shown', () => {
      component.onCheckboxToggle(all[0]);
      dataSource.goToPage(1);
      component.onMasterCheckboxToggle(true);
      expect(ids(component.selectedRows())).toEqual(['a', 'c', 'd']);
      expect(component.isAllLoadedSelected()).toBe(true);

      component.onMasterCheckboxToggle(false);
      expect(ids(component.selectedRows())).toEqual(['a']);
    });

    it('only auto-selects newly shown pages for a database-wide selection', () => {
      component.onMasterCheckboxToggle(true);
      dataSource.goToPage(1);
      expect(ids(component.selectedRows())).toEqual(['a', 'b']);

      component.onSelectAll();
      dataSource.goToPage(0);
      dataSource.goToPage(1);
      expect(ids(component.selectedRows())).toEqual(['a', 'b', 'c', 'd']);
    });

    it('clears the selection on every page with the clear selection button', () => {
      component.onCheckboxToggle(all[0]);
      dataSource.goToPage(1);
      fixture.detectChanges();

      (fixture.nativeElement.querySelector('.clear-selection') as HTMLElement).click();
      fixture.detectChanges();

      expect(component.selectedRows()).toEqual([]);
      expect(component.isSelectionMode()).toBe(false);
      expect(fixture.nativeElement.querySelector('.clear-selection')).toBeNull();
    });

    it('keeps the selection when the page changes but drops it for a new query', () => {
      const query = new BehaviorSubject('first');
      const queried = ListDataSource.fromQuery({
        query$: query,
        load: (_query, range) => of(...all.slice(range.offset, range.offset + range.pageSize)),
        destroyRef: fixture.componentRef.injector.get(DestroyRef),
      });
      component.dataSource = queried;
      queried.setPaging(2);
      component.onCheckboxToggle(all[0]);
      queried.goToPage(1);
      expect(ids(component.selectedRows())).toEqual(['a']);

      query.next('second');
      expect(component.selectedRows()).toEqual([]);
    });
  });

  describe('paging controls', () => {
    const rows = Array.from({length: 120}, (_, n) => n + 1).map(n => new ConfigObject({id: String(n), kind: Kind.SEED, meta: new Meta({name: `Seed ${n}`})}));

    beforeEach(() => {
      localStorage.clear();
      component.dataSource = ListDataSource.fromQuery({
        query$: of('query'),
        load: (_query, range) => of(...rows.slice(range.offset, range.offset + range.pageSize)),
        destroyRef: fixture.componentRef.injector.get(DestroyRef),
      });
      component.length = 120;
    });

    afterEach(() => localStorage.clear());

    it('offers no paging mode toggle or paginator unless paging is allowed', () => {
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('[aria-label="List mode"]')).toBeNull();
      expect(fixture.nativeElement.querySelector('mat-paginator')).toBeNull();
    });

    it('keeps the paginator clear of an end inset set by the host', () => {
      fixture.componentRef.setInput('allowPaging', true);
      fixture.detectChanges();
      (fixture.nativeElement.querySelectorAll('[aria-label="List mode"] button')[1] as HTMLElement).click();
      fixture.detectChanges();
      const paginator = fixture.nativeElement.querySelector('mat-paginator') as HTMLElement;
      // jsdom does not resolve var(), so only the reference to the host's inset is checked.
      expect(getComputedStyle(paginator).paddingInlineEnd).toContain('--config-list-end-inset');
    });

    it('switches between endless scroll and pages and remembers the choice', () => {
      fixture.componentRef.setInput('allowPaging', true);
      fixture.detectChanges();
      expect(component.dataSource.currentPaging).toBeNull();
      const modeButtons = () => [...fixture.nativeElement.querySelectorAll('[aria-label="List mode"] button')] as HTMLElement[];
      expect(modeButtons().map(button => button.getAttribute('aria-pressed'))).toEqual(['true', 'false']);
      expect(fixture.nativeElement.querySelector('mat-paginator')).toBeNull();

      modeButtons()[1].click();
      fixture.detectChanges();
      expect(modeButtons().map(button => button.getAttribute('aria-pressed'))).toEqual(['false', 'true']);
      expect(component.dataSource.currentPaging).toEqual({pageIndex: 0, pageSize: 50});
      expect(fixture.nativeElement.querySelector('mat-paginator')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.load-more-sentinel')).not.toBeNull();
      expect(localStorage.getItem('configList.pagingMode')).toBe('pages');

      modeButtons()[0].click();
      fixture.detectChanges();
      expect(component.dataSource.currentPaging).toBeNull();
      expect(fixture.nativeElement.querySelector('mat-paginator')).toBeNull();
    });

    it('moves between pages and changes the page size from the paginator', () => {
      fixture.componentRef.setInput('allowPaging', true);
      component.setPagingMode('pages');
      fixture.detectChanges();

      component.onPage({pageIndex: 1, pageSize: 50, previousPageIndex: 0, length: 120});
      expect(component.dataSource.currentPaging).toEqual({pageIndex: 1, pageSize: 50});

      component.onPage({pageIndex: 1, pageSize: 25, previousPageIndex: 1, length: 120});
      expect(component.dataSource.currentPaging).toEqual({pageIndex: 0, pageSize: 25});
      expect(localStorage.getItem('configList.pageSize')).toBe('25');
    });
  });
});

function horizontalCenter(element: Element): number {
  const {left, width} = element.getBoundingClientRect();
  return left + width / 2;
}
