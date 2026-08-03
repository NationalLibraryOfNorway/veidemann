import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {BehaviorSubject, EMPTY, of, Subscription} from 'rxjs';

import {AuthService, ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, Kind, ListRange} from '../../../../shared/models';
import {ConfigQuery} from '../../../../shared/func';
import {ConfigService} from '../../../../shared/services';
import {OptionsService} from '../../services';
import {ConfigurationsComponent} from './configurations.component';

describe('ConfigurationsComponent query loading', () => {
  let fixture: ComponentFixture<ConfigurationsComponent>;
  let component: ConfigurationsComponent;
  let queryParams: BehaviorSubject<ParamMap>;
  let kindParams: BehaviorSubject<ParamMap>;
  let lengthSubscription: Subscription;

  const search = vi.fn((query: ConfigQuery, range: ListRange) => {
    void query;
    void range;
    return EMPTY;
  });
  const count = vi.fn((query: ConfigQuery) => {
    void query;
    return of(0);
  });
  const update = vi.fn((config: ConfigObject) => of(config));
  const save = vi.fn((config: ConfigObject) => of(config));
  const deleteConfig = vi.fn((config: ConfigObject) => {
    void config;
    return of(true);
  });
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
    save.mockClear();
    deleteConfig.mockClear();
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
            update,
            save,
            delete: deleteConfig,
            loading$: of(false),
          }
        },
        {
          provide: Router,
          useValue: {navigate: vi.fn(() => Promise.resolve(true)), events: EMPTY}
        },
        {provide: MatDialog, useValue: dialog},
        {provide: OptionsService, useValue: {options$: of({})}},
        {provide: ControllerApiService, useValue: {}},
        {provide: AbilityServiceSignal, useValue: {can: () => false}},
        {provide: AuthService, useValue: {}},
        {provide: ErrorService, useValue: {dispatch: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
      ]
    })
      .overrideComponent(ConfigurationsComponent, {set: {template: ''}})
      .compileComponents();

    fixture = TestBed.createComponent(ConfigurationsComponent);
    component = fixture.componentInstance;
    lengthSubscription = component.length$.subscribe();
    fixture.detectChanges();
    await fixture.whenStable();
  });

  afterEach(() => lengthSubscription.unsubscribe());

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
  });

  it('reloads once per mutation and recounts only for additions and deletions', () => {
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
});
