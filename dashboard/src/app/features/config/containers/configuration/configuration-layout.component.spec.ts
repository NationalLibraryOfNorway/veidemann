import {AsyncPipe, Location} from '@angular/common';
import {NO_ERRORS_SCHEMA, signal} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, Router} from '@angular/router';
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
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
} from '../../../../shared/models';
import {ConfigService} from '../../../../shared/services';
import {JobStatusComponent} from '../../components';
import {CrawlExecutionStatusPipe, JobExecutionStatusPipe} from '../../pipe';
import {OptionsResolver, OptionsService} from '../../services';
import {RouterExtraService} from '../../services/router-extra.service';
import {ConfigurationComponent} from './configuration.component';

describe('ConfigurationComponent crawl-job layout', () => {
  async function createFixture(withRelatedConfiguration: boolean): Promise<ComponentFixture<ConfigurationComponent>> {
    const job = new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({
        crawlConfigRef: withRelatedConfiguration
          ? new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'})
          : null,
      }),
    });
    const latestExecution = new JobExecutionStatus({
      id: 'execution-1',
      jobId: job.id,
      state: JobExecutionState.FINISHED,
      startTime: '2026-08-09T10:00:00.000Z',
    });
    const canRead = vi.fn(() => true);

    await TestBed.configureTestingModule({
      imports: [ConfigurationComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(convertToParamMap({id: job.id})),
            parent: {paramMap: of(convertToParamMap({kind: 'crawljobs'}))},
          },
        },
        {
          provide: ConfigService,
          useValue: {
            get: (ref: ConfigRef) => of(ref.kind === Kind.CRAWLJOB
              ? job
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
          },
        },
        {provide: AbilityServiceSignal, useValue: {can: signal(false)}},
        {provide: ReportApiService, useValue: {getLastJobStatus: () => of(latestExecution)}},
        {provide: ControllerApiService, useValue: {}},
        {provide: SnackBarService, useValue: {}},
        {provide: RouterExtraService, useValue: {getCurrentUrl: () => '', getPreviousUrl: () => ''}},
        {provide: Location, useValue: {}},
        {provide: Router, useValue: {events: EMPTY, navigate: vi.fn(() => Promise.resolve(true))}},
        {provide: MatDialog, useValue: {open: vi.fn(), closeAll: vi.fn()}},
      ],
    })
      .overrideComponent(ConfigurationComponent, {
        set: {
          imports: [
            AsyncPipe,
            CrawlExecutionStatusPipe,
            JobExecutionStatusPipe,
            JobStatusComponent,
          ],
          schemas: [NO_ERRORS_SCHEMA],
        },
      })
      .compileComponents();

    const fixture = TestBed.createComponent(ConfigurationComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    return fixture;
  }

  it('places the latest execution in the aside before related configurations', async () => {
    const fixture = await createFixture(true);
    const primary = fixture.nativeElement.querySelector('.primary-pane') as HTMLElement;
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;
    const sectionTitles = [...aside.querySelectorAll(':scope > section.context-section > h2')]
      .map((heading: HTMLElement) => heading.textContent.trim());

    expect(primary.querySelector('app-config-job-execution-status')).toBeNull();
    expect(sectionTitles).toEqual(['Latest job execution', 'Related configurations']);
    expect(aside.querySelector('app-config-job-execution-status')).not.toBeNull();
  });

  it('keeps the aside when the latest execution is its only supporting content', async () => {
    const fixture = await createFixture(false);
    const aside = fixture.nativeElement.querySelector('.supporting-pane') as HTMLElement;

    expect(aside).not.toBeNull();
    expect(aside.textContent).toContain('Latest job execution');
    expect(aside.textContent).not.toContain('Related configurations');
  });
});
