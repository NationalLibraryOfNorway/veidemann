import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {DestroyRef, ElementRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {BehaviorSubject, of, Subject} from 'rxjs';
import {ConfigListComponent} from './config-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, Kind, Label, ListDataSource, Meta} from '../../../../shared/models';
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

  it('renders a loaded-row master checkbox only for multi-select lists', async () => {
    const row = new ConfigObject({id: 'one', meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    expect(header.querySelector('.master-selection-control')).not.toBeNull();
    expect(header.querySelector('.master-selection-control').classList).toContain('mat-mdc-list-item-icon');
    expect(await master.getAriaLabel()).toBe('Select all loaded configurations');
    expect(await master.isChecked()).toBe(false);
    expect(await master.isIndeterminate()).toBe(false);

    fixture.componentRef.setInput('multiSelect', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.list-header')).toBeNull();
  });

  it('uses checked and indeterminate master states and renders actions beside it', async () => {
    const first = new ConfigObject({id: 'one', meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));

    component.onCheckboxToggle(first);
    fixture.detectChanges();
    expect(await master.isChecked()).toBe(false);
    expect(await master.isIndeterminate()).toBe(true);
    const header = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const masterElement = header.querySelector('.master-selection-control') as HTMLElement;
    const actions = header.querySelector('.selection-actions') as HTMLElement;
    expect(masterElement.compareDocumentPosition(actions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(actions.classList).toContain('mat-mdc-list-item-title');

    await master.check();
    expect(component.selection.selected).toEqual([first, second]);
    expect(await master.isChecked()).toBe(true);
    expect(await master.isIndeterminate()).toBe(false);

    await master.uncheck();
    expect(component.selection.selected).toEqual([]);
    expect(fixture.nativeElement.querySelector('.selection-actions')).toBeNull();
  });

  it('renders the locale-formatted result count as trailing list metadata', () => {
    const first = new ConfigObject({id: 'one'});
    const second = new ConfigObject({id: 'two'});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(first, second),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.length = 3842;
    fixture.detectChanges();

    const count = fixture.nativeElement.querySelector('.result-count') as HTMLElement;
    expect(count.classList).toContain('mat-mdc-list-item-meta');
    expect(count.textContent.replace(/\s+/g, ' ').trim()).toBe('Showing 2 of 3,842 items');
  });

  it('automatically selects appended rows while loaded-row selection is active', async () => {
    const rows = new Subject<ConfigObject>();
    const first = new ConfigObject({id: 'one', kind: component.Kind.SEED, meta: new Meta({name: 'One'})});
    const second = new ConfigObject({id: 'two', kind: component.Kind.SEED, meta: new Meta({name: 'Two'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => rows,
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const selections: ConfigObject[][] = [];
    component.selectedChange.subscribe(value => selections.push([...value]));
    rows.next(first);
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));

    await master.check();
    rows.next(second);
    fixture.detectChanges();

    expect(component.selection.selected).toEqual([first, second]);
    expect(selections.at(-1)).toEqual([first, second]);
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
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));
    await master.check();
    fixture.detectChanges();

    let chip = fixture.nativeElement.querySelector('.database-selection-chip') as HTMLElement;
    expect(fixture.nativeElement.querySelector('.selection-summary').textContent.replace(/\s+/g, ' ').trim())
      .toContain('All 1 seeds on this page are selected.');
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

    await master.check();
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

  it('does not offer database selection when all matching rows are loaded', async () => {
    const row = new ConfigObject({id: 'one', meta: new Meta({name: 'One'})});
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.length = 1;
    fixture.detectChanges();
    const master = await loader.getHarness(MatCheckboxHarness.with({selector: '.master-selection-control'}));

    await master.check();
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
  ])('uses the route-specific plural label for kind %s', (kind, label) => {
    component.selectedRows.set([new ConfigObject({id: 'selected', kind})]);

    expect(component.configTypePluralLabel()).toBe(label);
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

  it('adds a configured external chip without changing label search behavior', () => {
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
    const external = fixture.nativeElement.querySelector('.external-label-link') as HTMLAnchorElement;
    expect(external.textContent.trim()).toBe('Brønnøysundregistrene');
    expect(external.getAttribute('href'))
      .toBe('https://virksomhet.brreg.no/nb/oppslag/enheter/976%20029%2F100');
    expect(external.getAttribute('target')).toBe('_blank');
    expect(external.getAttribute('rel')).toBe('noopener noreferrer');

    external.click();
    expect(selectedLabels).toEqual([]);
    searchChip.click();
    expect(selectedLabels).toEqual([label]);
  });

  it.each([
    {text: '', urlTemplate: 'https://example.com/{value}'},
    {text: 'Registry', urlTemplate: 'https://example.com/static'},
    {text: 'Registry', urlTemplate: 'javascript:{value}'},
  ])('ignores invalid external label configuration %#', link => {
    appConfig.labelLinks = {owner: link};
    const row = new ConfigObject({
      id: 'labeled',
      meta: new Meta({
        name: 'Labeled configuration',
        labelList: [new Label({key: 'owner', value: 'archive'})],
      }),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.external-label-link')).toBeNull();
    expect(fixture.nativeElement.querySelectorAll('mat-chip')).toHaveLength(1);
  });

  it('preserves row navigation, disabled styling, and bulk selection behavior', () => {
    const enabled = new ConfigObject({
      id: 'enabled',
      meta: new Meta({name: 'Enabled'}),
    });
    const disabled = new ConfigObject({
      id: 'disabled',
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
    expect(rows[0].classList).not.toContain('row-disabled');
    expect(rows[1].classList).toContain('row-disabled');

    rows[0].click();
    rows[0].dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    (rows[0].querySelector('.primary-detail-link') as HTMLElement).click();
    expect(rowClicks).toEqual([enabled, enabled, enabled]);

    component.onMasterCheckboxToggle(true);
    expect(selections.at(-1)).toEqual([enabled, disabled]);
    expect(component.isAllLoadedSelected()).toBe(true);
    component.onSelectAll();
    expect(selectAllCount).toBe(1);
    expect(component.allSelected()).toBe(true);
    component.onDeselectAll();
    expect(component.selection.selected).toEqual([]);
    expect(component.allSelected()).toBe(false);
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
