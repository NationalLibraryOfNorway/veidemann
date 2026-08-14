import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {By} from '@angular/platform-browser';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {BehaviorSubject, EMPTY, from, Observable, of, Subject} from 'rxjs';

import {AuthService, ControllerApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {BrowserScriptType, ConfigObject, Kind, Label, ListRange, Role, RobotsPolicy} from '../../../../shared/models';
import {ConfigQuery} from '../../../../shared/func';
import {ConfigService} from '../../../../shared/services';
import {OptionsService} from '../../services';
import {ConfigListComponent} from '../../components';
import {ConfigurationsComponent} from './configurations.component';
import {MultiUpdateDialogComponent} from '../../components/multi-update-dialog/multi-update-dialog.component';

describe('ConfigurationsComponent query loading', () => {
  let fixture: ComponentFixture<ConfigurationsComponent>;
  let component: ConfigurationsComponent;
  let queryParams: BehaviorSubject<ParamMap>;
  let kindParams: BehaviorSubject<ParamMap>;

  const search = vi.fn((query: ConfigQuery, range: ListRange): Observable<ConfigObject> => {
    void query;
    void range;
    return EMPTY;
  });
  const count = vi.fn((query: ConfigQuery) => {
    void query;
    return of(0);
  });
  const update = vi.fn((config: ConfigObject) => of(config));
  const updateWithTemplate = vi.fn(() => of(2));
  const startUpdateWithTemplate = vi.fn(() => of('task-123'));
  const save = vi.fn((config: ConfigObject) => of(config));
  const deleteConfig = vi.fn((config: ConfigObject) => {
    void config;
    return of(true);
  });
  const can = vi.fn(() => false);
  const openSnackBar = vi.fn();
  const navigate = vi.fn<Router['navigate']>().mockResolvedValue(true);
  const dialog = {
    open: vi.fn(() => ({afterClosed: () => of(true)})),
    closeAll: vi.fn(),
  };

  beforeEach(async () => {
    queryParams = new BehaviorSubject(convertToParamMap({}));
    kindParams = new BehaviorSubject(convertToParamMap({kind: 'seed'}));
    search.mockClear();
    count.mockClear();
    update.mockClear();
    updateWithTemplate.mockClear();
    startUpdateWithTemplate.mockClear();
    save.mockClear();
    deleteConfig.mockClear();
    can.mockReset();
    can.mockReturnValue(false);
    openSnackBar.mockClear();
    navigate.mockClear();
    dialog.open.mockClear();

    await TestBed.configureTestingModule({
      imports: [ConfigurationsComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            queryParamMap: queryParams,
            parent: {paramMap: kindParams},
          }
        },
        {
          provide: ConfigService,
          useValue: {
            search,
            count,
            get: () => of(null),
            update,
            updateWithTemplate,
            startUpdateWithTemplate,
            save,
            delete: deleteConfig,
            loading$: of(false),
          }
        },
        {
          provide: Router,
          useValue: {navigate, events: EMPTY}
        },
        {provide: MatDialog, useValue: dialog},
        {provide: OptionsService, useValue: {options$: of({})}},
        {provide: ControllerApiService, useValue: {}},
        {provide: AbilityServiceSignal, useValue: {can}},
        {
          provide: AuthService,
          useValue: {
            canRead: () => false,
            canUpdate: () => false,
            canCreate: () => false,
            canRunCrawl: () => false,
          }
        },
        {provide: SnackBarService, useValue: {openSnackBar}},
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigurationsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('performs one initial search and one atomic search for a parameter-map change', async () => {
    expect(search).toHaveBeenCalledTimes(1);
    expect(count).toHaveBeenCalledTimes(1);

    queryParams.next(convertToParamMap({q: 'term', p: '2', s: '50'}));
    await fixture.whenStable();

    expect(search).toHaveBeenCalledTimes(2);
    expect(count).toHaveBeenCalledTimes(2);
    expect(search.mock.calls.at(-1)[0]).toEqual(expect.objectContaining({
      term: 'term',
    }));
    expect(search.mock.calls.at(-1)[1]).toEqual({offset: 0, pageSize: 100});
  });

  it('shows one locale-formatted, type-specific loaded and database count above the list', async () => {
    const rows = Array.from({length: 240}, (_, index) => new ConfigObject({id: `${index}`, kind: Kind.SEED}));
    search.mockClear();
    count.mockClear();
    search.mockReturnValueOnce(from(rows));
    count.mockReturnValueOnce(of(3842));

    queryParams.next(convertToParamMap({q: 'counted'}));
    await fixture.whenStable();
    fixture.detectChanges();

    const summary = fixture.nativeElement.querySelector('.result-count') as HTMLElement;
    const header = fixture.nativeElement.querySelector('.list-header') as HTMLElement;
    const scroll = fixture.nativeElement.querySelector('app-config-list .scroll') as HTMLElement;
    const list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    expect(summary.textContent.replace(/\s+/g, ' ').trim()).toBe('240 of 3,842 seeds');
    expect(header.compareDocumentPosition(scroll) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(list.totalLength()).toBe(3842);
    expect(count).toHaveBeenCalledTimes(1);
  });

  it('hides stale totals while a new database count is pending', async () => {
    const firstCount = new Subject<number>();
    count.mockReturnValueOnce(firstCount);
    queryParams.next(convertToParamMap({q: 'pending'}));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.result-count')).toBeNull();

    firstCount.next(12);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.result-count').textContent)
      .toContain('0 of 12 seeds');

    const secondCount = new Subject<number>();
    count.mockReturnValueOnce(secondCount);
    queryParams.next(convertToParamMap({q: 'next-pending'}));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.result-count')).toBeNull();
  });

  it('updates the loaded count as rows stream without recounting the database', async () => {
    const rows = new Subject<ConfigObject>();
    search.mockClear();
    count.mockClear();
    search.mockReturnValueOnce(rows);
    count.mockReturnValueOnce(of(10));
    queryParams.next(convertToParamMap({q: 'streamed'}));
    await fixture.whenStable();

    rows.next(new ConfigObject({id: 'one'}));
    rows.next(new ConfigObject({id: 'two'}));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.result-count').textContent)
      .toContain('2 of 10 seeds');
    expect(count).toHaveBeenCalledTimes(1);
  });

  it('uses the shared configuration list and result count for collections and role mappings', async () => {
    kindParams.next(convertToParamMap({kind: 'collection'}));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.result-count').textContent)
      .toContain('0 of 0 collections');
    expect((fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent).multiSelect)
      .toBe(true);

    kindParams.next(convertToParamMap({kind: 'rolemapping'}));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.result-count').textContent)
      .toContain('0 of 0 role mappings');
    expect(fixture.debugElement.query(By.directive(ConfigListComponent))).not.toBeNull();
    expect(fixture.nativeElement.querySelector('app-rolemapping-list')).toBeNull();
  });

  it('opens mass update in a responsive viewport-safe dialog', () => {
    const selected = new ConfigObject({kind: Kind.SEED});
    component['selectedConfigs'] = [selected];
    component.isAllSelected = false;
    dialog.open.mockReturnValueOnce({afterClosed: () => EMPTY});

    component.onEditSelected();

    expect(dialog.open).toHaveBeenCalledWith(MultiUpdateDialogComponent, expect.objectContaining({
      data: expect.objectContaining({configObject: selected, allSelected: false}),
      width: '720px',
      maxWidth: 'calc(100vw - 32px)',
      maxHeight: 'calc(100dvh - 32px)',
      autoFocus: 'first-tabbable',
      restoreFocus: true,
    }));
  });

  it('starts all-database updates in the background without reloading', () => {
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    component.isAllSelected = true;
    search.mockClear();
    count.mockClear();

    component.onUpdateMulti({updateTemplate, pathList: ['meta.label-']});

    expect(startUpdateWithTemplate).toHaveBeenCalledWith(updateTemplate, ['meta.label-']);
    expect(updateWithTemplate).not.toHaveBeenCalled();
    expect(search).not.toHaveBeenCalled();
    expect(count).not.toHaveBeenCalled();
    expect(openSnackBar).toHaveBeenCalledWith(expect.stringContaining('task-123'));
  });

  it('keeps explicitly selected multi-updates synchronous', () => {
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    component.isAllSelected = false;
    component['selectedConfigs'] = [new ConfigObject({id: 'seed-1', kind: Kind.SEED})];

    component.onUpdateMulti({updateTemplate, pathList: ['seed.disabled']});

    expect(updateWithTemplate).toHaveBeenCalledWith(updateTemplate, ['seed.disabled'], ['seed-1']);
    expect(startUpdateWithTemplate).not.toHaveBeenCalled();
  });

  it('loads and serializes the BrowserScript type route filter', async () => {
    kindParams.next(convertToParamMap({kind: 'browserscript'}));
    await fixture.whenStable();
    search.mockClear();
    count.mockClear();

    queryParams.next(convertToParamMap({script_type: BrowserScriptType.ON_LOAD.toString()}));
    await fixture.whenStable();

    expect(search.mock.calls.at(-1)[0].browserScriptType).toBe(BrowserScriptType.ON_LOAD);
    expect(count.mock.calls.at(-1)[0].browserScriptType).toBe(BrowserScriptType.ON_LOAD);

    component.onQueryChange({...component.query(), browserScriptType: BrowserScriptType.SCOPE_CHECK});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({
        p: null,
        s: null,
        script_type: BrowserScriptType.SCOPE_CHECK,
      }),
    }));
  });

  it('loads, clears, serializes, and restores the robot-policy route filter', async () => {
    kindParams.next(convertToParamMap({kind: 'politenessconfig'}));
    queryParams.next(convertToParamMap({robots_policy: RobotsPolicy.CUSTOM_IF_MISSING.toString()}));
    await fixture.whenStable();
    expect(search.mock.calls.at(-1)[0].robotsPolicy).toBe(RobotsPolicy.CUSTOM_IF_MISSING);
    expect(count.mock.calls.at(-1)[0].robotsPolicy).toBe(RobotsPolicy.CUSTOM_IF_MISSING);

    component.onQueryChange({...component.query(), robotsPolicy: null});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({robots_policy: null}),
    }));

    queryParams.next(convertToParamMap({robots_policy: RobotsPolicy.OBEY_ROBOTS_CLASSIC.toString()}));
    await fixture.whenStable();
    expect(component.query().robotsPolicy).toBe(RobotsPolicy.OBEY_ROBOTS_CLASSIC);
  });

  it('loads and serializes the role mapping role filter', async () => {
    kindParams.next(convertToParamMap({kind: 'rolemapping'}));
    queryParams.next(convertToParamMap({role: Role.OPERATOR.toString()}));
    await fixture.whenStable();

    expect(search.mock.calls.at(-1)[0].role).toBe(Role.OPERATOR);
    expect(count.mock.calls.at(-1)[0].role).toBe(Role.OPERATOR);

    component.onQueryChange({...component.query(), role: Role.CURATOR});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({role: Role.CURATOR}),
    }));
  });

  it('replaces the text query with an exact label search and preserves filters and sort', async () => {
    queryParams.next(convertToParamMap({
      q: 'old search',
      p: '3',
      s: '50',
      sort: 'name:desc',
      entity_id: 'entity-1',
      disabled: 'true',
      crawl_job_id: 'job-1',
    }));
    await fixture.whenStable();
    navigate.mockClear();

    component.onFilterByLabel(new Label({key: 'owner', value: 'archive'}));

    expect(navigate).toHaveBeenCalledTimes(1);
    const [, options] = navigate.mock.calls[0];
    expect(options.queryParamsHandling).toBe('merge');
    expect(options.queryParams).toEqual(expect.objectContaining({
      p: null,
      s: null,
      q: 'label:owner:archive',
      entity_id: 'entity-1',
      disabled: true,
      crawl_job_id: ['job-1'],
    }));
    expect(options.queryParams['sort']).toBeUndefined();
  });

  it('merges state changes with an applied label search', async () => {
    queryParams.next(convertToParamMap({
      q: 'example label:owner:archive',
      sort: 'name:asc',
      entity_id: 'entity-1',
    }));
    await fixture.whenStable();
    navigate.mockClear();

    component.onDisabledFilterChange(true);

    expect(navigate).toHaveBeenCalledOnce();
    const [, options] = navigate.mock.calls[0];
    expect(options.queryParamsHandling).toBe('merge');
    expect(options.queryParams).toEqual(expect.objectContaining({
      q: 'example label:owner:archive',
      disabled: true,
      entity_id: 'entity-1',
    }));
  });

  it('enables state controls only for Seed and Crawl Job and ordering for every list', async () => {
    let list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    expect(list.showStateFilter).toBe(true);
    expect(list.showOrderControl).toBe(true);

    kindParams.next(convertToParamMap({kind: 'crawljobs'}));
    await fixture.whenStable();
    fixture.detectChanges();
    list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    expect(list.showStateFilter).toBe(true);
    expect(list.showOrderControl).toBe(true);

    kindParams.next(convertToParamMap({kind: 'collection'}));
    await fixture.whenStable();
    fixture.detectChanges();
    list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    expect(list.showStateFilter).toBe(false);
    expect(list.showOrderControl).toBe(true);
  });

  it('serializes supported ordering and removes the sort parameter for default order', () => {
    component.onSort({active: 'name', direction: 'asc'});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({sort: 'name:asc'}),
    }));

    component.onSort({active: 'lastModified', direction: 'desc'});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({sort: 'lastModified:desc'}),
    }));

    component.onSort({active: '', direction: ''});
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({sort: null}),
    }));
  });

  it('removes an applied label filter while preserving the name query', async () => {
    queryParams.next(convertToParamMap({
      q: 'example label:owner:archive',
      entity_id: 'entity-1',
    }));
    await fixture.whenStable();
    navigate.mockClear();

    component.onRemoveFilter({
      key: 'labelSelector',
      value: 'owner:archive',
      label: 'owner:archive',
    });

    expect(navigate).toHaveBeenCalledTimes(1);
    const [, options] = navigate.mock.calls[0];
    expect(options.queryParams).toEqual(expect.objectContaining({
      q: 'example',
      entity_id: 'entity-1',
    }));
  });

  it('removes one Crawl Job filter while preserving other active filters', async () => {
    queryParams.next(convertToParamMap({
      q: 'example label:owner:archive',
      entity_id: 'entity-1',
      crawl_job_id: ['job-1', 'job-2'],
    }));
    await fixture.whenStable();
    navigate.mockClear();

    component.onRemoveFilter({
      key: 'crawlJobIdList',
      value: 'job-1',
      label: 'Daily job',
      icon: 'work',
    });

    expect(navigate).toHaveBeenCalledTimes(1);
    const [, options] = navigate.mock.calls[0];
    expect(options.queryParams).toEqual(expect.objectContaining({
      q: 'example label:owner:archive',
      entity_id: 'entity-1',
      crawl_job_id: ['job-2'],
    }));
  });

  it('searches once without recounting for page and sort changes or unrelated parameters', async () => {
    search.mockClear();
    count.mockClear();

    queryParams.next(convertToParamMap({p: '1', sort: 'name:desc', unrelated: 'first'}));
    await fixture.whenStable();

    expect(search).toHaveBeenCalledTimes(1);
    expect(count).not.toHaveBeenCalled();

    queryParams.next(convertToParamMap({p: '1', sort: 'name:desc', unrelated: 'second'}));
    await fixture.whenStable();

    expect(search).toHaveBeenCalledTimes(1);
    expect(count).not.toHaveBeenCalled();
  });

  it('loads a new kind once without replacing the data source', async () => {
    const dataSource = component.dataSource;
    search.mockClear();
    count.mockClear();

    kindParams.next(convertToParamMap({kind: 'collection'}));
    await fixture.whenStable();

    expect(component.dataSource).toBe(dataSource);
    expect(search).toHaveBeenCalledTimes(1);
    expect(count).toHaveBeenCalledTimes(1);
    expect(search.mock.calls[0][0].kind).toBe(Kind.COLLECTION);
    fixture.detectChanges();
    const list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    expect(list.multiSelect).toBe(true);
  });

  it('recounts additions and deletions but not unfiltered updates', () => {
    const config = new ConfigObject({id: 'config', kind: Kind.SEED});
    search.mockClear();
    count.mockClear();

    component.onUpdateConfig(config);
    expect(search).toHaveBeenCalledTimes(1);
    expect(count).not.toHaveBeenCalled();

    search.mockClear();
    component.onSaveConfig(config);
    expect(search).toHaveBeenCalledTimes(1);
    expect(count).toHaveBeenCalledTimes(1);

    search.mockClear();
    count.mockClear();
    component.onDeleteConfig(config);
    expect(search).toHaveBeenCalledTimes(1);
    expect(count).toHaveBeenCalledTimes(1);
  });

  it('opens confirmation and run dialogs as Escape-dismissible', () => {
    dialog.open.mockReturnValue({afterClosed: () => EMPTY});
    const seed = new ConfigObject({id: 'seed', kind: Kind.SEED});

    component.onDeleteConfig(seed);
    component.onDeleteConfigObjects([seed]);
    component.onRunCrawl(seed);
    component.onRunCrawlSelected([seed]);

    expect(dialog.open).toHaveBeenCalledTimes(4);
    for (const call of dialog.open.mock.calls as unknown as [unknown, {disableClose?: boolean}][]) {
      expect(call[1]).toEqual(expect.objectContaining({disableClose: false}));
    }
  });

  it('recounts an update when active filters can change the matching total', async () => {
    const config = new ConfigObject({id: 'config', kind: Kind.SEED});
    queryParams.next(convertToParamMap({q: 'filtered'}));
    await fixture.whenStable();
    search.mockClear();
    count.mockClear();

    component.onUpdateConfig(config);

    expect(search).toHaveBeenCalledTimes(1);
    expect(count).toHaveBeenCalledTimes(1);
  });

  it('clears the container selection when a new query resets the list', async () => {
    const selected = new ConfigObject({id: 'selected', kind: Kind.SEED});
    const state = component as unknown as {selectedConfigs: ConfigObject[]};
    component.onSelectedChange([selected]);
    component.onSelectAll();

    queryParams.next(convertToParamMap({q: 'next'}));
    await fixture.whenStable();

    expect(state.selectedConfigs).toEqual([]);
    expect(component.isAllSelected).toBe(false);
  });

  it('renders seed actions when a loaded seed is selected', async () => {
    const seed = new ConfigObject({id: 'selected-seed', kind: Kind.SEED});
    search.mockReturnValueOnce(of(seed));

    queryParams.next(convertToParamMap({q: 'selectable'}));
    await fixture.whenStable();
    fixture.detectChanges();

    const list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    list.onCheckboxToggle(seed);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(component['selectedConfigs']).toEqual([seed]);
    const bulkAction = fixture.nativeElement.querySelector(
      '[aria-label="Start crawl for selected seeds"]'
    ) as HTMLButtonElement;
    expect(bulkAction).not.toBeNull();
    expect(bulkAction.classList).toContain('filled-icon-button');
    expect(bulkAction.querySelector('mat-icon')?.textContent.trim()).toBe('play_arrow');
  });

  it('renders the create action as a FAB outside the filter toolbar', async () => {
    can.mockReturnValue(true);
    kindParams.next(convertToParamMap({kind: 'collection'}));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const toolbar = fixture.nativeElement.querySelector('.filter-toolbar') as HTMLElement;
    const activeFilters = fixture.nativeElement.querySelector('app-active-filter-chips') as HTMLElement;
    const fab = fixture.nativeElement.querySelector('.create-fab') as HTMLButtonElement;

    expect(toolbar.getAttribute('aria-label')).toBe('Configuration filters');
    expect(toolbar.querySelector('app-config-query')).not.toBeNull();
    expect(toolbar.compareDocumentPosition(activeFilters) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(toolbar.querySelector('.create-fab')).toBeNull();
    expect(fab).not.toBeNull();
    expect(fab.hasAttribute('mat-fab')).toBe(true);
    expect(fab.getAttribute('aria-label')).toBe('Create a new configuration');
    expect(fixture.nativeElement.querySelector('mat-drawer')).toBeNull();
  });
});
