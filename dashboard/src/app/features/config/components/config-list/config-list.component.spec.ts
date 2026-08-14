import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {DestroyRef, ElementRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {MatMenuHarness} from '@angular/material/menu/testing';
import {BehaviorSubject, of, Subject} from 'rxjs';
import {ConfigListComponent} from './config-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, Kind, Label, ListDataSource, Meta, Role, RoleMapping} from '../../../../shared/models';
import {AppConfig} from '../../../../app.config';

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;
  let loader: HarnessLoader;
  let appConfig: AppConfig;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        ConfigListComponent,
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

  it('renders role mapping identities as titles and roles as subtitles', () => {
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
    const selectionButtons = fixture.nativeElement.querySelectorAll('.selection-entry-control') as NodeListOf<HTMLButtonElement>;
    expect(Array.from(titles, title => title.textContent.trim()))
      .toEqual(['curator@example.test', 'operators']);
    expect(Array.from(subtitles, subtitle => subtitle.textContent.trim()))
      .toEqual(['CURATOR, READONLY', 'OPERATOR, SYSTEM']);
    expect(Array.from(selectionButtons, button => button.getAttribute('aria-label')))
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

  it('shows a kind icon in normal mode and reveals the master checkbox after selection starts', async () => {
    const row = new ConfigObject({id: 'one', kind: Kind.SEED, meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();
    const entry = fixture.nativeElement.querySelector('.selection-entry-control') as HTMLButtonElement;
    expect(entry.textContent.trim()).toBe('link');
    expect(entry.getAttribute('aria-label')).toBe('Select One');

    entry.click();
    fixture.detectChanges();

    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(header.querySelector('.master-selection-control')).not.toBeNull();
    expect(await master.getAriaLabel()).toBe('Select all loaded configurations');
    expect(await master.isChecked()).toBe(true);
    expect(await master.isIndeterminate()).toBe(false);
    expect(fixture.nativeElement.querySelector('.selection-entry-control')).toBeNull();
    const masterControl = header.querySelector('.master-selection-control') as HTMLElement;
    const rowControl = fixture.nativeElement.querySelector('.item-row .selection-control') as HTMLElement;
    expect(Math.abs(horizontalCenter(masterControl) - horizontalCenter(rowControl))).toBeLessThanOrEqual(1);

    fixture.componentRef.setInput('multiSelect', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();
    expect(fixture.nativeElement.querySelector('.config-kind-icon').textContent.trim()).toBe('link');
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

  it('renders the order control before deselectable state icon buttons', () => {
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
    const states: (boolean | null)[] = [];
    component.disabledFilterChange.subscribe(value => states.push(value));
    fixture.detectChanges();

    const headerControls = fixture.nativeElement.querySelector('.list-header-controls') as HTMLElement;
    const stateFilter = headerControls.querySelector('.state-filter') as HTMLElement;
    const orderControl = headerControls.querySelector('.order-control') as HTMLElement;
    const resultCount = fixture.nativeElement.querySelector('.result-count') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.selection-leading')).toBeNull();
    expect(orderControl.compareDocumentPosition(stateFilter) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(headerControls.compareDocumentPosition(resultCount) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    const orderIcon = orderControl.querySelector('mat-icon') as HTMLElement;
    const rowIcon = fixture.nativeElement.querySelector('.selection-entry-control mat-icon') as HTMLElement;
    expect(Math.abs(horizontalCenter(orderIcon) - horizontalCenter(rowIcon))).toBeLessThanOrEqual(1);

    const [deactivated, active] = [...stateFilter.querySelectorAll('button')] as HTMLButtonElement[];
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

    active.click();
    expect(states.at(-1)).toBe(false);
    fixture.componentRef.setInput('disabledFilter', false);
    fixture.detectChanges();
    expect(active.getAttribute('aria-pressed')).toBe('true');
    expect(deactivated.getAttribute('aria-pressed')).toBe('false');

    deactivated.click();
    expect(states.at(-1)).toBe(true);
    fixture.componentRef.setInput('disabledFilter', true);
    fixture.detectChanges();
    expect(active.getAttribute('aria-pressed')).toBe('false');
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');

    deactivated.click();
    expect(states.at(-1)).toBeNull();
  });

  it('offers every supported order and emits the corresponding sort', async () => {
    component.showOrderControl = true;
    component.length = 0;
    const sorts: {active: string; direction: string}[] = [];
    component.sort.subscribe(value => sorts.push(value));
    fixture.detectChanges();

    const button = await loader.getHarness(MatButtonHarness.with({
      selector: '[data-testid="configuration-order"]',
      appearance: 'text',
      iconName: 'sort',
    }));
    expect(await button.getAppearance()).toBe('text');
    const orderButton = fixture.nativeElement.querySelector('.order-control') as HTMLButtonElement;
    expect(orderButton.tagName).toBe('BUTTON');
    expect(orderButton.classList).toContain('order-control-icon-only');
    expect(orderButton.querySelector('.order-label')).toBeNull();
    expect(orderButton.querySelector('.order-menu-indicator')).toBeNull();
    expect(fixture.nativeElement.querySelector('.list-header-controls mat-form-field')).toBeNull();
    expect(fixture.nativeElement.querySelector('.list-header-controls mat-select')).toBeNull();

    const menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
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
    expect(orderButton.classList).not.toContain('order-control-icon-only');
    const orderLabel = orderButton.querySelector('.order-label') as HTMLElement;
    const orderIndicator = orderButton.querySelector('.order-menu-indicator') as HTMLElement;
    expect(orderLabel.textContent?.trim()).toBe('Name: A–Z');
    expect(orderLabel.compareDocumentPosition(orderIndicator) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await menu.open();
    await menu.clickItem({text: 'Name: Z–A'});
    expect(sorts.at(-1)).toEqual({active: 'name', direction: 'desc'});
    fixture.componentRef.setInput('sortDirection', 'desc');
    fixture.detectChanges();

    await menu.open();
    await menu.clickItem({text: 'Last modified: newest first'});
    expect(sorts.at(-1)).toEqual({active: 'lastModified', direction: 'desc'});
    fixture.componentRef.setInput('sortActive', 'lastModified');
    fixture.detectChanges();

    await menu.open();
    await menu.clickItem({text: 'Last modified: oldest first'});
    expect(sorts.at(-1)).toEqual({active: 'lastModified', direction: 'asc'});
    fixture.componentRef.setInput('sortDirection', 'asc');
    fixture.detectChanges();

    await menu.open();
    await menu.clickItem({text: 'Default order'});
    expect(sorts.at(-1)).toEqual({active: '', direction: ''});
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

    const [deactivated] = [...states.querySelectorAll('button')] as HTMLButtonElement[];
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');

    component.onDeselectAll();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.selection-leading')).toBeNull();
    expect(fixture.nativeElement.querySelector('.order-control')?.textContent).toContain('Name: A–Z');
    expect(deactivated.getAttribute('aria-pressed')).toBe('true');
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
    const first = new ConfigObject({id: 'one', kind: Kind.SEED});
    const second = new ConfigObject({id: 'two', kind: Kind.SEED});
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

  it.each([
    [Kind.CRAWLENTITY, 'business'],
    [Kind.SEED, 'link'],
    [Kind.CRAWLJOB, 'work'],
    [Kind.CRAWLSCHEDULECONFIG, 'schedule'],
    [Kind.CRAWLCONFIG, 'settings_system_daydream'],
    [Kind.COLLECTION, 'collections_bookmark'],
    [Kind.BROWSERCONFIG, 'web'],
    [Kind.BROWSERSCRIPT, 'web_asset'],
    [Kind.POLITENESSCONFIG, 'sentiment_very_satisfied'],
    [Kind.CRAWLHOSTGROUPCONFIG, 'group_work'],
    [Kind.ROLEMAPPING, 'people'],
  ])('uses the configuration navigation icon for kind %s', (kind, icon) => {
    expect(component.configKindIcon(new ConfigObject({kind}))).toBe(icon);
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

  it('aligns title and supporting text against the leading control', () => {
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
    expect(Array.from(text).every(element => getComputedStyle(element).top === '-4px')).toBe(true);
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

    (fixture.nativeElement.querySelector('.selection-entry-control') as HTMLElement).click();
    fixture.detectChanges();

    const chip = fixture.nativeElement.querySelector('.label-region mat-chip') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.primary-detail-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.external-label-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.action-region')).toBeNull();
    expect(chip.getAttribute('role')).not.toBe('button');
    expect(chip.hasAttribute('tabindex')).toBe(false);
  });

  it('toggles rows by pointer and keyboard in selection mode and Escape clears selection', () => {
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

    (fixture.nativeElement.querySelector('.selection-entry-control') as HTMLElement).click();
    fixture.detectChanges();
    const rows = fixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    rows[1].click();
    expect(component.selection.selected).toEqual([first, second]);
    expect(navigated).toEqual([]);

    rows[1].dispatchEvent(new KeyboardEvent('keydown', {key: ' ', bubbles: true, cancelable: true}));
    expect(component.selection.selected).toEqual([first]);
    rows[1].dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true}));
    expect(component.selection.selected).toEqual([first, second]);

    const escape = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
    rows[0].dispatchEvent(escape);
    fixture.detectChanges();
    expect(escape.defaultPrevented).toBe(true);
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-entry-control')).not.toBeNull();
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

  it('keeps Collections non-selectable with a decorative kind icon', () => {
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

    expect(fixture.nativeElement.querySelector('.selection-entry-control')).toBeNull();
    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();
    expect(fixture.nativeElement.querySelector('.config-kind-icon').textContent.trim())
      .toBe('collections_bookmark');
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
});

function horizontalCenter(element: HTMLElement): number {
  const rect = element.getBoundingClientRect();
  return rect.left + rect.width / 2;
}
