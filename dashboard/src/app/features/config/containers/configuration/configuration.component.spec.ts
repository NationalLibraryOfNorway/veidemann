import {Location} from '@angular/common';
import {ErrorHandler, signal, WritableSignal} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, NEVER, of, throwError} from 'rxjs';

import {AuthService, ControllerApiService, SnackBarService} from '../../../../core';
import {
  Annotation,
  BrowserConfig,
  ConfigObject,
  ConfigRef,
  CrawlJob,
  Kind,
  Label,
  Meta,
  Seed,
} from '../../../../shared/models';
import {ConfigService} from '../../../../shared/services';
import {OptionsResolver, OptionsService} from '../../services';
import {RouterExtraService} from '../../services/router-extra.service';
import {ConfigurationComponent} from './configuration.component';
import {AbilityServiceSignal} from '@casl/angular';
import {AppConfig} from '../../../../app.config';

describe('ConfigurationComponent route loading', () => {
  let fixture: ComponentFixture<ConfigurationComponent>;
  let idParams: BehaviorSubject<ParamMap>;
  let kindParams: BehaviorSubject<ParamMap>;
  let options: BehaviorSubject<{browserScripts?: ConfigObject[]; crawlJobs?: ConfigObject[]}>;
  let canRead: ReturnType<typeof vi.fn>;
  let loadOptions: ReturnType<typeof vi.fn>;
  let dialogOpen: ReturnType<typeof vi.fn>;
  let locationBack: ReturnType<typeof vi.fn>;
  let navigate: ReturnType<typeof vi.fn>;
  let seedReadPermission: WritableSignal<boolean>;
  let appConfig: AppConfig;

  const get = vi.fn((ref: ConfigRef) => of(new ConfigObject({id: ref.id, kind: ref.kind})));
  const search = vi.fn(() => of<ConfigObject>());
  const getScriptAnnotations = vi.fn(() => of([]));
  const save = vi.fn((configObject: ConfigObject) => of(configObject));
  const update = vi.fn((configObject: ConfigObject) => of(configObject));
  const deleteConfig = vi.fn(() => of(true));
  const move = vi.fn(() => of(1));

  beforeEach(async () => {
    idParams = new BehaviorSubject(convertToParamMap({id: 'entity-1'}));
    kindParams = new BehaviorSubject(convertToParamMap({kind: 'entity'}));
    options = new BehaviorSubject({});
    get.mockClear();
    search.mockReset();
    search.mockReturnValue(of());
    getScriptAnnotations.mockClear();
    save.mockClear();
    update.mockClear();
    deleteConfig.mockClear();
    move.mockClear();
    canRead = vi.fn(() => true);
    seedReadPermission = signal(true);
    loadOptions = vi.fn(() => of({}));
    dialogOpen = vi.fn(() => ({componentInstance: {}, afterClosed: () => EMPTY}));
    locationBack = vi.fn();
    navigate = vi.fn(() => Promise.resolve(true));

    await TestBed.configureTestingModule({
      imports: [ConfigurationComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {paramMap: idParams, parent: {paramMap: kindParams}},
        },
        {
          provide: ConfigService,
          useValue: {get, getScriptAnnotations, save, update, delete: deleteConfig, move, search, loading$: of(false)},
        },
        {provide: OptionsService, useValue: {options$: options, next: vi.fn()}},
        {provide: OptionsResolver, useValue: {load: loadOptions}},
        {provide: AuthService, useValue: {canRead}},
        {
          provide: AbilityServiceSignal,
          useValue: {
            can: (action: string, subject: string) =>
              action === 'read' && subject === Kind[Kind.SEED] && seedReadPermission(),
          },
        },
        {provide: ControllerApiService, useValue: {}},
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
        {
          provide: RouterExtraService,
          useValue: {getCurrentUrl: () => '', getPreviousUrl: () => ''},
        },
        {provide: Location, useValue: {back: locationBack}},
        {provide: Router, useValue: {events: EMPTY, navigate}},
        {provide: MatDialog, useValue: {open: dialogOpen, closeAll: vi.fn()}},
      ],
    })
      .overrideComponent(ConfigurationComponent, {set: {template: ''}})
      .compileComponents();

    appConfig = TestBed.inject(AppConfig);
    appConfig.labelLinks = {};
    fixture = TestBed.createComponent(ConfigurationComponent);
  });

  it('loads immediately and reloads when the selected route changes', async () => {
    const loaded: ConfigObject[] = [];
    fixture.componentInstance.configObject$.subscribe(config => loaded.push(config));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(get).toHaveBeenLastCalledWith(expect.objectContaining({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
    }));

    idParams.next(convertToParamMap({id: 'entity-2'}));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(get).toHaveBeenCalledTimes(2);
    expect(get).toHaveBeenLastCalledWith(expect.objectContaining({
      id: 'entity-2',
      kind: Kind.CRAWLENTITY,
    }));
    expect(loaded.at(-1)).toEqual(expect.objectContaining({id: 'entity-2'}));
  });

  it('resolves matching label links for the detail context aside', () => {
    appConfig.labelLinks = {
      owner: {text: 'Owner registry', urlTemplate: 'https://example.com/owners/{value}'},
      ignored: {text: 'Ignored', urlTemplate: 'javascript:{value}'},
    };
    const configObject = new ConfigObject({
      meta: new Meta({labelList: [
        new Label({key: 'owner', value: 'national archive'}),
        new Label({key: 'ignored', value: 'unsafe'}),
      ]}),
    });

    expect(fixture.componentInstance.labelLinksFor(configObject)).toEqual([{
      text: 'Owner registry',
      href: 'https://example.com/owners/national%20archive',
    }]);
  });

  it('loads only the first seed page for an entity supporting list', async () => {
    search.mockReturnValue(of(
      new ConfigObject({id: 'seed-a', kind: Kind.SEED, meta: new Meta({name: 'Alpha'})}),
      new ConfigObject({id: 'seed-z', kind: Kind.SEED, meta: new Meta({name: 'Zulu'})}),
    ));
    search.mockClear();
    fixture.componentInstance.entitySeedDataSource.reload();
    fixture.detectChanges();
    const seeds = await firstValueFrom(fixture.componentInstance.entitySeedDataSource.rows$);

    expect(seeds).toEqual([
      expect.objectContaining({id: 'seed-a'}),
      expect.objectContaining({id: 'seed-z'}),
    ]);
    expect(search).toHaveBeenCalledWith(
      expect.objectContaining({kind: Kind.SEED, entityId: 'entity-1'}),
      {offset: 0, pageSize: 100},
    );
  });

  it('loads the initial entity seed page without requiring an explicit reload', async () => {
    fixture.destroy();
    search.mockReturnValue(of(
      new ConfigObject({id: 'seed-initial', kind: Kind.SEED, meta: new Meta({name: 'Initial seed'})}),
    ));
    search.mockClear();
    fixture = TestBed.createComponent(ConfigurationComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const rows = await firstValueFrom(fixture.componentInstance.entitySeedDataSource.rows$);
    expect(rows).toEqual([expect.objectContaining({id: 'seed-initial'})]);
    expect(search).toHaveBeenCalledWith(
      expect.objectContaining({kind: Kind.SEED, entityId: 'entity-1'}),
      {offset: 0, pageSize: 100},
    );
  });

  it('starts loading seeds from the route before entity details finish loading', async () => {
    fixture.destroy();
    get.mockReturnValueOnce(NEVER);
    search.mockReturnValue(of(
      new ConfigObject({id: 'seed-before-entity', kind: Kind.SEED, meta: new Meta({name: 'Early seed'})}),
    ));
    search.mockClear();
    fixture = TestBed.createComponent(ConfigurationComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(search).toHaveBeenCalledWith(
      expect.objectContaining({kind: Kind.SEED, entityId: 'entity-1'}),
      {offset: 0, pageSize: 100},
    );
    await expect(firstValueFrom(fixture.componentInstance.entitySeedDataSource.rows$))
      .resolves.toEqual([expect.objectContaining({id: 'seed-before-entity'})]);
  });

  it('loads the initial seed page when seed-read permission becomes available', async () => {
    fixture.destroy();
    seedReadPermission.set(false);
    search.mockReturnValue(of(
      new ConfigObject({id: 'seed-after-auth', kind: Kind.SEED, meta: new Meta({name: 'Authorized seed'})}),
    ));
    search.mockClear();
    fixture = TestBed.createComponent(ConfigurationComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    expect(search).not.toHaveBeenCalled();

    seedReadPermission.set(true);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(search).toHaveBeenCalledWith(
      expect.objectContaining({kind: Kind.SEED, entityId: 'entity-1'}),
      {offset: 0, pageSize: 100},
    );
    await expect(firstValueFrom(fixture.componentInstance.entitySeedDataSource.rows$))
      .resolves.toEqual([expect.objectContaining({id: 'seed-after-auth'})]);
  });

  it('reloads the bounded entity seed list when its state chip changes', async () => {
    search.mockClear();

    fixture.componentInstance.onEntitySeedStatusChange(true);
    await fixture.whenStable();

    expect(search).toHaveBeenLastCalledWith(
      expect.objectContaining({kind: Kind.SEED, entityId: 'entity-1', disabled: true}),
      {offset: 0, pageSize: 100},
    );
  });

  it('reloads seed details after a successful entity move', async () => {
    const component = fixture.componentInstance;
    component.configObject$.subscribe();
    fixture.detectChanges();
    await fixture.whenStable();
    get.mockClear();
    const seed = new ConfigObject({id: 'seed-1', kind: Kind.SEED});
    const entityRef = new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-2'});

    component.onMoveSeed({seed, entityRef});
    await fixture.whenStable();

    expect(move).toHaveBeenCalledWith(seed, entityRef);
    expect(get).toHaveBeenCalled();
  });

  it('loads target-kind options before cloning a related configuration', () => {
    const rotationPolicies = [1];
    loadOptions.mockReturnValue(of({rotationPolicies}));

    fixture.componentInstance.onClone(new ConfigObject({id: 'collection-1', kind: Kind.COLLECTION}));

    expect(loadOptions).toHaveBeenCalledWith(Kind.COLLECTION);
    expect(dialogOpen).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
      data: expect.objectContaining({options: expect.objectContaining({rotationPolicies})}),
    }));
  });

  it('opens related Edit and Clone dialogs in supporting-context mode', () => {
    const configObject = new ConfigObject({id: 'collection-1', kind: Kind.COLLECTION});
    const openDialog = vi.spyOn(fixture.componentInstance, 'onCreateConfigWithDialog');

    fixture.componentInstance.onEditRelatedConfig(configObject);
    fixture.componentInstance.onCloneRelatedConfig(configObject);

    expect(openDialog).toHaveBeenNthCalledWith(1, configObject, 'related');
    expect(openDialog).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({id: '', kind: Kind.COLLECTION}),
      'related',
    );
  });

  it('updates a related configuration without navigating away', () => {
    const related = new ConfigObject({id: 'collection-1', kind: Kind.COLLECTION});
    dialogOpen.mockReturnValue({componentInstance: {}, afterClosed: () => of(related)});

    fixture.componentInstance.onEditRelatedConfig(related);

    expect(update).toHaveBeenCalledWith(related);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('keeps the entity detail open after creating a seed', () => {
    const entity = new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY});
    const seed = new ConfigObject({
      kind: Kind.SEED,
      seed: new Seed({entityRef: ConfigObject.toConfigRef(entity)}),
    });
    dialogOpen.mockReturnValue({componentInstance: {move: NEVER}, afterClosed: () => of(seed)});

    fixture.componentInstance.onCreateSeedFromEntity(entity);

    expect(save).toHaveBeenCalledWith(seed);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('does not navigate when a related configuration dialog is canceled', () => {
    const related = new ConfigObject({id: 'collection-1', kind: Kind.COLLECTION});
    dialogOpen.mockReturnValue({componentInstance: {}, afterClosed: () => of(undefined)});

    fixture.componentInstance.onEditRelatedConfig(related);

    expect(update).not.toHaveBeenCalled();
    expect(navigate).not.toHaveBeenCalled();
  });

  it('deletes a related configuration without leaving the current detail', () => {
    dialogOpen.mockReturnValue({componentInstance: {}, afterClosed: () => of(true)});
    const related = new ConfigObject({id: 'collection-1', kind: Kind.COLLECTION});

    fixture.componentInstance.onDeleteRelatedConfig(related);

    expect(deleteConfig).toHaveBeenCalledWith(related);
    expect(locationBack).not.toHaveBeenCalled();
    expect(navigate).not.toHaveBeenCalled();
  });

  it('opens confirmation and run dialogs as Escape-dismissible and accepts an empty result', () => {
    dialogOpen.mockReturnValue({componentInstance: {}, afterClosed: () => of(undefined)});
    const seed = new ConfigObject({id: 'seed', kind: Kind.SEED});
    fixture.componentInstance.options = {crawlJobs: []};

    fixture.componentInstance.onDeleteConfig(seed);
    fixture.componentInstance.onRunCrawl(seed);

    expect(dialogOpen).toHaveBeenCalledTimes(2);
    for (const call of dialogOpen.mock.calls as unknown as [unknown, {disableClose?: boolean}][]) {
      expect(call[1]).toEqual(expect.objectContaining({disableClose: false}));
    }
  });

  it('opens a seed run dialog with the related crawljob preselected', () => {
    const seed = new ConfigObject({id: 'seed-1', kind: Kind.SEED});
    const crawlJob = new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB});
    fixture.componentInstance.options = {crawlJobs: [crawlJob]};

    fixture.componentInstance.onRunSeedInCrawlJob(seed, crawlJob);

    expect(dialogOpen).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
      data: {configObject: seed, jobRefId: crawlJob.id, crawlJobs: [crawlJob]},
    }));
  });

  it('navigates saved related configurations through their own absolute kind path', () => {
    fixture.componentInstance.onSaveConfig(new ConfigObject({id: 'crawl-config-2', kind: Kind.CRAWLCONFIG}));

    expect(navigate).toHaveBeenCalledWith(['/config', 'crawlconfig', 'crawl-config-2']);
  });

  it('loads direct related configurations in order and isolates failed references', async () => {
    idParams.next(convertToParamMap({id: 'job-1'}));
    kindParams.next(convertToParamMap({kind: 'crawljobs'}));
    get.mockImplementation((ref: ConfigRef) => {
      if (ref.kind === Kind.CRAWLJOB) {
        return of(new ConfigObject({
          id: ref.id,
          kind: ref.kind,
          crawlJob: new CrawlJob({
            scheduleRef: new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG, id: 'schedule-1'}),
            crawlConfigRef: new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'}),
          }),
        }));
      }
      if (ref.id === 'schedule-1') {
        return throwError(() => new Error('missing schedule'));
      }
      return of(new ConfigObject({id: ref.id, kind: ref.kind}));
    });

    const related = firstValueFrom(fixture.componentInstance.relatedConfigs$);
    fixture.detectChanges();

    await expect(related).resolves.toEqual([
      expect.objectContaining({
        descriptor: expect.objectContaining({
          ref: expect.objectContaining({id: 'schedule-1'}), role: 'schedule', source: 'direct',
        }),
        configObject: null,
        unavailable: true,
      }),
      expect.objectContaining({
        descriptor: expect.objectContaining({
          ref: expect.objectContaining({id: 'crawl-config-1'}), role: 'crawl-config', source: 'direct',
        }),
        configObject: expect.objectContaining({id: 'crawl-config-1'}),
        unavailable: false,
      }),
    ]);
  });

  it('does not request related configurations without read permission', async () => {
    idParams.next(convertToParamMap({id: 'job-1'}));
    kindParams.next(convertToParamMap({kind: 'crawljobs'}));
    canRead.mockImplementation(kind => kind !== Kind.CRAWLSCHEDULECONFIG);
    get.mockImplementation((ref: ConfigRef) => ref.kind === Kind.CRAWLJOB
      ? of(new ConfigObject({
        id: ref.id,
        kind: ref.kind,
        crawlJob: new CrawlJob({
          scheduleRef: new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG, id: 'schedule-1'}),
          crawlConfigRef: new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'}),
        }),
      }))
      : of(new ConfigObject({id: ref.id, kind: ref.kind})));

    const related = firstValueFrom(fixture.componentInstance.relatedConfigs$);
    fixture.detectChanges();

    await expect(related).resolves.toEqual([
      expect.objectContaining({descriptor: expect.objectContaining({
        ref: expect.objectContaining({id: 'crawl-config-1'}),
      })}),
    ]);
    expect(get).not.toHaveBeenCalledWith(expect.objectContaining({id: 'schedule-1'}));
  });

  it('loads explicitly and implicitly selected BrowserScripts as related configurations', async () => {
    idParams.next(convertToParamMap({id: 'browser-config-1'}));
    kindParams.next(convertToParamMap({kind: 'browserconfig'}));
    const browserScripts = [
      new ConfigObject({id: 'explicit-script', kind: Kind.BROWSERSCRIPT}),
      new ConfigObject({
        id: 'implicit-script',
        kind: Kind.BROWSERSCRIPT,
        meta: new Meta({labelList: [new Label({key: 'profile', value: 'default'})]}),
      }),
      new ConfigObject({
        id: 'unrelated-script',
        kind: Kind.BROWSERSCRIPT,
        meta: new Meta({labelList: [new Label({key: 'profile', value: 'other'})]}),
      }),
    ];
    options.next({browserScripts});
    get.mockImplementation((ref: ConfigRef) => ref.kind === Kind.BROWSERCONFIG
      ? of(new ConfigObject({
        id: ref.id,
        kind: ref.kind,
        browserConfig: new BrowserConfig({
          scriptRefList: [new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'explicit-script'})],
          scriptSelectorList: ['profile:default'],
        }),
      }))
      : of(browserScripts.find(script => script.id === ref.id)));

    const related = firstValueFrom(fixture.componentInstance.relatedConfigs$);
    fixture.detectChanges();

    await expect(related).resolves.toEqual([
      expect.objectContaining({
        descriptor: expect.objectContaining({
          ref: expect.objectContaining({id: 'explicit-script'}), source: 'direct',
        }),
        configObject: expect.objectContaining({id: 'explicit-script'}),
      }),
      expect.objectContaining({
        descriptor: expect.objectContaining({
          ref: expect.objectContaining({id: 'implicit-script'}), source: 'selector',
        }),
        configObject: expect.objectContaining({id: 'implicit-script'}),
      }),
    ]);
    expect(get).not.toHaveBeenCalledWith(expect.objectContaining({id: 'unrelated-script'}));
  });

  it('does not request implicitly selected BrowserScripts without read permission', async () => {
    idParams.next(convertToParamMap({id: 'browser-config-1'}));
    kindParams.next(convertToParamMap({kind: 'browserconfig'}));
    options.next({
      browserScripts: [new ConfigObject({
        id: 'implicit-script',
        kind: Kind.BROWSERSCRIPT,
        meta: new Meta({labelList: [new Label({key: 'profile', value: 'default'})]}),
      })],
    });
    canRead.mockImplementation(kind => kind !== Kind.BROWSERSCRIPT);
    get.mockImplementation((ref: ConfigRef) => of(new ConfigObject({
      id: ref.id,
      kind: ref.kind,
      browserConfig: new BrowserConfig({scriptSelectorList: ['profile:default']}),
    })));

    const related = firstValueFrom(fixture.componentInstance.relatedConfigs$);
    fixture.detectChanges();

    await expect(related).resolves.toEqual([]);
    expect(get).not.toHaveBeenCalledWith(expect.objectContaining({id: 'implicit-script'}));
  });

  it('loads unique script annotation keys as editor suggestions', async () => {
    idParams.next(convertToParamMap({id: 'job-1'}));
    kindParams.next(convertToParamMap({kind: 'crawljobs'}));
    get.mockImplementation((ref: ConfigRef) => of(new ConfigObject({
      id: ref.id,
      kind: ref.kind,
      meta: new Meta({name: 'Daily crawl'}),
    })));
    getScriptAnnotations.mockReturnValue(of([
      new Annotation({key: 'scope_maxHopsFromSeed', value: '2'}),
      new Annotation({key: 'scope_altSeeds', value: ''}),
      new Annotation({key: 'scope_altSeeds', value: 'alt.example'}),
    ]));

    const suggestions = firstValueFrom(fixture.componentInstance.annotationSuggestions$);
    fixture.detectChanges();

    await expect(suggestions).resolves.toEqual(['scope_altSeeds', 'scope_maxHopsFromSeed']);
    expect(getScriptAnnotations).toHaveBeenCalledWith('job-1', undefined);
  });

  it('loads seed-specific suggestions for each referenced crawljob', async () => {
    const jobRef = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
    idParams.next(convertToParamMap({id: 'seed-1'}));
    kindParams.next(convertToParamMap({kind: 'seed'}));
    options.next({
      crawlJobs: [new ConfigObject({
        id: jobRef.id,
        kind: jobRef.kind,
        meta: new Meta({name: 'Daily crawl'}),
      })],
    });
    get.mockImplementation((ref: ConfigRef) => of(new ConfigObject({
      id: ref.id,
      kind: ref.kind,
      seed: new Seed({jobRefList: [jobRef]}),
    })));
    getScriptAnnotations.mockReturnValue(of([
      new Annotation({key: 'scope_altSeeds', value: 'alt.example'}),
    ]));

    const suggestions = firstValueFrom(fixture.componentInstance.annotationSuggestions$);
    fixture.detectChanges();

    await expect(suggestions).resolves.toEqual(['scope_altSeeds']);
    expect(getScriptAnnotations).toHaveBeenCalledWith('job-1', 'seed-1');
  });
});
