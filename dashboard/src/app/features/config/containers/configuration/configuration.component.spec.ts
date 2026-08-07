import {Location} from '@angular/common';
import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, of, throwError} from 'rxjs';

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

describe('ConfigurationComponent route loading', () => {
  let fixture: ComponentFixture<ConfigurationComponent>;
  let idParams: BehaviorSubject<ParamMap>;
  let kindParams: BehaviorSubject<ParamMap>;
  let options: BehaviorSubject<{browserScripts?: ConfigObject[]; crawlJobs?: ConfigObject[]}>;
  let canRead: ReturnType<typeof vi.fn>;
  let loadOptions: ReturnType<typeof vi.fn>;
  let dialogOpen: ReturnType<typeof vi.fn>;
  let navigate: ReturnType<typeof vi.fn>;

  const get = vi.fn((ref: ConfigRef) => of(new ConfigObject({id: ref.id, kind: ref.kind})));
  const getScriptAnnotations = vi.fn(() => of([]));
  const save = vi.fn((configObject: ConfigObject) => of(configObject));
  const move = vi.fn(() => of(1));

  beforeEach(async () => {
    idParams = new BehaviorSubject(convertToParamMap({id: 'entity-1'}));
    kindParams = new BehaviorSubject(convertToParamMap({kind: 'entity'}));
    options = new BehaviorSubject({});
    get.mockClear();
    getScriptAnnotations.mockClear();
    move.mockClear();
    canRead = vi.fn(() => true);
    loadOptions = vi.fn(() => of({}));
    dialogOpen = vi.fn(() => ({componentInstance: {}, afterClosed: () => EMPTY}));
    navigate = vi.fn(() => Promise.resolve(true));

    await TestBed.configureTestingModule({
      imports: [ConfigurationComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {paramMap: idParams, parent: {paramMap: kindParams}},
        },
        {provide: ConfigService, useValue: {get, getScriptAnnotations, save, move, loading$: of(false)}},
        {provide: OptionsService, useValue: {options$: options, next: vi.fn()}},
        {provide: OptionsResolver, useValue: {load: loadOptions}},
        {provide: AuthService, useValue: {canRead}},
        {provide: ControllerApiService, useValue: {}},
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
        {provide: RouterExtraService, useValue: {}},
        {provide: Location, useValue: {}},
        {provide: Router, useValue: {events: EMPTY, navigate}},
        {provide: MatDialog, useValue: {open: dialogOpen, closeAll: vi.fn()}},
      ],
    })
      .overrideComponent(ConfigurationComponent, {set: {template: ''}})
      .compileComponents();

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
        ref: expect.objectContaining({id: 'schedule-1'}),
        configObject: null,
        unavailable: true,
      }),
      expect.objectContaining({
        ref: expect.objectContaining({id: 'crawl-config-1'}),
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
      expect.objectContaining({ref: expect.objectContaining({id: 'crawl-config-1'})}),
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
        ref: expect.objectContaining({id: 'explicit-script'}),
        configObject: expect.objectContaining({id: 'explicit-script'}),
      }),
      expect.objectContaining({
        ref: expect.objectContaining({id: 'implicit-script'}),
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
