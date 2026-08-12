import {AsyncPipe, Location} from '@angular/common';
import {NO_ERRORS_SCHEMA, signal} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, Router, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {EMPTY, of} from 'rxjs';

import {
  AuthService,
  ControllerApiService,
  ReportApiService,
  SnackBarService,
} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  ConfigRef,
  CrawlExecutionState,
  CrawlExecutionStatus,
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
  Label,
  Meta,
  Seed,
} from '../../../../shared/models';
import {ConfigService} from '../../../../shared/services';
import {CrawlExecutionStatusComponent, EntitySeedContextComponent, JobStatusComponent} from '../../components';
import {CrawlExecutionStatusPipe, JobExecutionStatusPipe} from '../../pipe';
import {OptionsResolver, OptionsService} from '../../services';
import {RouterExtraService} from '../../services/router-extra.service';
import {ConfigurationComponent} from './configuration.component';
import {ConfigPath} from '../../func';
import {DetailHeaderComponent} from '../../../../shared/components';
import {AppConfig} from '../../../../app.config';
import {ConfigLabelLinksComponent} from '../../components/config-label-links/config-label-links.component';

describe('ConfigurationComponent crawl-job layout', () => {
  async function createFixture(
    kind: Kind,
    withRelatedConfiguration: boolean,
    canReadExecution = true,
  ): Promise<ComponentFixture<ConfigurationComponent>> {
    const configObject = kind === Kind.CRAWLJOB
      ? new ConfigObject({
        id: 'job-1',
        kind,
        crawlJob: new CrawlJob({
          crawlConfigRef: withRelatedConfiguration
            ? new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'})
            : null,
        }),
      })
      : kind === Kind.SEED ? new ConfigObject({
        id: 'seed-1',
        kind,
        meta: new Meta({name: 'https://example.com/seed'}),
        seed: new Seed({
          entityRef: withRelatedConfiguration
            ? new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-1'})
            : null,
        }),
      }) : new ConfigObject({
        id: `config-${kind}`,
        kind,
        meta: new Meta({
          name: `Configuration ${kind}`,
          labelList: kind === Kind.CRAWLENTITY ? [new Label({key: 'owner', value: 'archive'})] : [],
        }),
      });
    const latestExecution = new JobExecutionStatus({
      id: 'execution-1',
      jobId: configObject.id,
      state: JobExecutionState.FINISHED,
      startTime: '2026-08-09T10:00:00.000Z',
    });
    const latestCrawlExecution = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      jobId: 'crawl-job-1',
      state: CrawlExecutionState.FINISHED,
      startTime: '2026-08-09T10:00:00.000Z',
    });
    const crawlJob = new ConfigObject({
      id: latestCrawlExecution.jobId,
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Daily crawl'}),
    });
    const canRead = vi.fn((subject: string) =>
      !['jobexecution', 'crawlexecution'].includes(subject) || canReadExecution);

    await TestBed.configureTestingModule({
      imports: [ConfigurationComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(convertToParamMap({id: configObject.id})),
            parent: {paramMap: of(convertToParamMap({kind: ConfigPath[kind]}))},
          },
        },
        {
          provide: ConfigService,
          useValue: {
            get: (ref: ConfigRef) => of(ref.id === configObject.id
              ? configObject
              : new ConfigObject({id: ref.id, kind: ref.kind})),
            getScriptAnnotations: () => of([]),
            search: () => EMPTY,
            loading$: of(false),
          },
        },
        {
          provide: OptionsService,
          useValue: {
            options$: of({
              browserScripts: [],
              crawlConfigs: [],
              crawlScheduleConfigs: [],
              crawlJobs: [crawlJob],
              scopeScripts: [],
            }),
          },
        },
        {provide: OptionsResolver, useValue: {}},
        {
          provide: AuthService,
          useValue: {
            canRead,
            canCreate: () => false,
            canUpdate: () => false,
            canDelete: () => false,
            canRunCrawl: () => false,
          },
        },
        {provide: AbilityServiceSignal, useValue: {can: signal(true)}},
        {
          provide: ReportApiService,
          useValue: {
            getLastJobStatus: () => of(latestExecution),
            getLastSeedStatus: () => of(latestCrawlExecution),
          },
        },
        {provide: ControllerApiService, useValue: {}},
        {provide: SnackBarService, useValue: {}},
        {provide: RouterExtraService, useValue: {getCurrentUrl: () => '', getPreviousUrl: () => ''}},
        {provide: Location, useValue: {}},
        {
          provide: Router,
          useValue: {
            events: EMPTY,
            navigate: vi.fn(() => Promise.resolve(true)),
            createUrlTree: vi.fn((commands: string[]) => ({commands})),
            serializeUrl: vi.fn(({commands}: {commands: string[]}) => commands.join('/')),
          },
        },
        {provide: MatDialog, useValue: {open: vi.fn(), closeAll: vi.fn()}},
      ],
    })
      .overrideComponent(ConfigurationComponent, {
        set: {
          imports: [
            AsyncPipe,
            CrawlExecutionStatusComponent,
            CrawlExecutionStatusPipe,
            ConfigLabelLinksComponent,
            DetailHeaderComponent,
            EntitySeedContextComponent,
            JobExecutionStatusPipe,
            JobStatusComponent,
            RouterLink,
          ],
          schemas: [NO_ERRORS_SCHEMA],
        },
      })
      .compileComponents();

    TestBed.inject(AppConfig).labelLinks = {
      owner: {text: 'Owner registry', urlTemplate: 'https://example.com/owners/{value}'},
    };
    const fixture = TestBed.createComponent(ConfigurationComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    return fixture;
  }

  it('places the latest execution in the primary pane before the editor', async () => {
    const fixture = await createFixture(Kind.CRAWLJOB, true);
    const primary = fixture.nativeElement.querySelector('.primary-pane') as HTMLElement;
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const summary = primary.querySelector('app-config-job-execution-status') as HTMLElement;
    const editor = primary.querySelector('.configuration-editor') as HTMLElement;
    expect(summary).not.toBeNull();
    expect(summary.compareDocumentPosition(editor) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(primary.querySelector('.configuration-relationships')).toBeNull();
    expect(primary.querySelector('app-detail-header app-config-shortcut-helpers[detailHeaderActions]')).not.toBeNull();
    const relatedSection = aside.querySelector('.context-section') as HTMLElement;
    const relatedHeading = relatedSection.querySelector('h2') as HTMLElement;
    expect(relatedHeading.textContent.trim()).toBe('Related configurations');
    expect(relatedSection.getAttribute('aria-labelledby')).toBe(relatedHeading.id);
    expect([...relatedSection.querySelectorAll('.related-context-group')]
      .every((group: HTMLElement) => {
        const heading = group.querySelector('h3') as HTMLElement;
        return !!heading.textContent.trim() && group.getAttribute('aria-labelledby') === heading.id;
      })).toBeTruthy();
    expect(aside.querySelector('app-config-label-links')).toBeNull();
    expect(aside.firstElementChild).toBe(relatedSection);
    expect(aside.querySelector('app-config-job-execution-status')).toBeNull();
    const asideStyle = getComputedStyle(aside);
    expect(asideStyle.gap).toBe(getComputedStyle(primary).gap);
    expect(asideStyle.getPropertyValue('--detail-pane-gap').trim()).toBe('32px');
    expect(getComputedStyle(aside.querySelector('.related-context-groups')).gap).toBe(asideStyle.gap);
    const relatedListStyle = getComputedStyle(aside.querySelector('.related-context-list'));
    expect(relatedListStyle.gap).toBe('16px');
    expect(relatedListStyle.borderTopStyle).toBe('solid');
    expect(relatedListStyle.paddingBlockStart).toBe('12px');
  });

  it('does not create an aside for the latest execution alone', async () => {
    const fixture = await createFixture(Kind.CRAWLJOB, false);
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const executionSection = fixture.nativeElement.querySelector('.execution-section') as HTMLElement;

    expect(aside).toBeNull();
    const heading = executionSection.querySelector('h2') as HTMLElement;
    const link = heading.querySelector('a') as HTMLAnchorElement;
    expect(executionSection.getAttribute('aria-labelledby')).toBe(heading.id);
    expect(heading.textContent.trim()).toBe('Latest job execution');
    expect(link.getAttribute('href')).toBe('/report/jobexecution/execution-1');
  });

  it('places the latest seed crawl execution in the primary pane', async () => {
    const fixture = await createFixture(Kind.SEED, true);
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const primary = fixture.nativeElement.querySelector('.primary-pane') as HTMLElement;
    const summary = primary.querySelector('app-config-crawl-execution-status .execution-summary') as HTMLElement;
    const executionSection = primary.querySelector('.execution-section') as HTMLElement;
    const editor = primary.querySelector('.configuration-editor') as HTMLElement;

    const relatedSection = aside.querySelector('.context-section') as HTMLElement;
    const relatedHeading = relatedSection.querySelector('h2') as HTMLElement;
    const groupHeading = relatedSection.querySelector('.related-context-group h3') as HTMLElement;
    expect(relatedHeading.textContent.trim()).toBe('Related configurations');
    expect(relatedSection.getAttribute('aria-labelledby')).toBe(relatedHeading.id);
    expect(groupHeading.textContent.trim()).not.toBe('');
    expect(groupHeading.parentElement.getAttribute('aria-labelledby')).toBe(groupHeading.id);
    expect(summary).not.toBeNull();
    expect(executionSection.compareDocumentPosition(editor) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(primary.querySelector('.configuration-relationships')).toBeNull();
    expect(primary.querySelector('app-detail-header app-config-shortcut-helpers[detailHeaderActions]')).not.toBeNull();
    expect([...summary.querySelectorAll('app-execution-metadata dt')]
      .map((term: HTMLElement) => term.textContent.trim())).toEqual(['Started', 'Finished', 'Crawl job']);
    expect(summary.querySelectorAll('app-execution-metadata dd')[2].textContent.trim()).toBe('Daily crawl');
    expect(summary.querySelector('mat-expansion-panel')).toBeNull();
  });

  it('renders the seed title as an external link', async () => {
    const fixture = await createFixture(Kind.SEED, false);
    const titleLink = fixture.nativeElement.querySelector(
      'app-detail-header.configuration-header h1 a',
    ) as HTMLAnchorElement;

    expect(titleLink.textContent.trim()).toBe('https://example.com/seed');
    expect(titleLink.href).toBe('https://example.com/seed');
    expect(titleLink.target).toBe('_blank');
    expect(titleLink.rel).toBe('noopener noreferrer');
  });

  it('does not create a seed aside for the latest crawl execution alone', async () => {
    const fixture = await createFixture(Kind.SEED, false);
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const executionSection = fixture.nativeElement.querySelector('.execution-section') as HTMLElement;

    expect(aside).toBeNull();
    const heading = executionSection.querySelector('h2') as HTMLElement;
    const link = heading.querySelector('a') as HTMLAnchorElement;
    expect(executionSection.getAttribute('aria-labelledby')).toBe(heading.id);
    expect(heading.textContent.trim()).toBe('Latest crawl execution');
    expect(link.getAttribute('href')).toBe('/report/crawlexecution/crawl-execution-1');
  });

  it.each([
    [Kind.CRAWLJOB, 'Latest job execution'],
    [Kind.SEED, 'Latest crawl execution'],
  ])('renders the %s execution heading as plain text without report permission', async (kind, title) => {
    const fixture = await createFixture(kind, false, false);
    const heading = fixture.nativeElement.querySelector('.execution-section h2') as HTMLElement;

    expect(heading.textContent.trim()).toBe(title);
    expect(heading.querySelector('a')).toBeNull();
  });

  it('shows resolved entity label links in the page supporting aside', async () => {
    const fixture = await createFixture(Kind.CRAWLENTITY, false);
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const labelLinks = aside.querySelector('app-config-label-links') as HTMLElement;
    const seedContext = aside.querySelector('app-entity-seed-context') as HTMLElement;
    const link = labelLinks.querySelector('a') as HTMLAnchorElement;

    expect(aside).not.toBeNull();
    expect(link.textContent.trim()).toBe('Owner registry');
    expect(link.href).toBe('https://example.com/owners/archive');
    expect(seedContext).not.toBeNull();
    expect([...aside.querySelectorAll('h2')]
      .map((heading: HTMLElement) => heading.textContent.trim())).toEqual(['Links', 'Seeds']);
    expect(labelLinks.compareDocumentPosition(seedContext) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it.each([
    Kind.CRAWLENTITY,
    Kind.SEED,
    Kind.CRAWLJOB,
    Kind.CRAWLSCHEDULECONFIG,
    Kind.CRAWLCONFIG,
    Kind.COLLECTION,
    Kind.BROWSERCONFIG,
    Kind.BROWSERSCRIPT,
    Kind.POLITENESSCONFIG,
    Kind.CRAWLHOSTGROUPCONFIG,
    Kind.ROLEMAPPING,
  ])('owns one page heading and no top-level editor card for kind %s', async kind => {
    const fixture = await createFixture(kind, false);
    const primary = fixture.nativeElement.querySelector('.primary-pane') as HTMLElement;
    const editor = primary.querySelector('.configuration-editor') as HTMLElement;

    expect(primary.querySelectorAll(':scope > app-detail-header.configuration-header h1').length).toBe(1);
    expect(editor).not.toBeNull();
    expect(editor.querySelector(':scope > mat-card')).toBeNull();
  });
});
