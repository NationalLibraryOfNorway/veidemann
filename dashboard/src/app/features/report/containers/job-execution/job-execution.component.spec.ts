import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {JobExecutionComponent} from './job-execution.component';
import {JobExecutionService} from '../../services';
import {ActivatedRoute, convertToParamMap} from '@angular/router';
import {of} from 'rxjs';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {provideNativeDateAdapter} from '@angular/material/core';
import {MatMenuHarness} from '@angular/material/menu/testing';
import {AbilityServiceSignal} from '@casl/angular';
import {ConfigObject, JobExecutionStatus, Meta} from '../../../../shared/models';

describe('JobExecutionComponent', () => {
  let component: JobExecutionComponent;
  let fixture: ComponentFixture<JobExecutionComponent>;
  let loader: HarnessLoader;

  const row = new JobExecutionStatus({id: 'execution-1', jobId: 'job-1'});

  const fakeActivatedRoute = {
    queryParamMap: of(convertToParamMap({})),
    snapshot: {
      data: {
        options: {}
      }
    },
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [JobExecutionComponent],
      declarations: [],
      providers: [
        ...provideCoreTesting,
        provideNativeDateAdapter(),
        {provide: ActivatedRoute, useValue: fakeActivatedRoute},
        {provide: AbilityServiceSignal, useValue: {can: () => true}},
        {
          provide: JobExecutionService,
          useValue: {
            search: () => of(row),
            getJob: () => of(new ConfigObject({meta: new Meta({name: 'Daily crawl'})})),
            loading$: of(false),
          }
        },
      ]
    })
      .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(JobExecutionComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    loader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('should create', async () => {
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('places its filter toolbar before the result list', () => {
    const filters = fixture.nativeElement.querySelector('.report-filter-toolbar') as HTMLElement;
    const results = fixture.nativeElement.querySelector('.report-result-content') as HTMLElement;

    expect(filters.compareDocumentPosition(results) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('offers only the crawl executions for this job execution action', async () => {
    const menu = await loader.getHarness(MatMenuHarness);
    await menu.open();
    const items = await menu.getItems();

    expect(items).toHaveLength(1);
    expect(await items[0].getText()).toContain('Go to crawl executions for this job execution');
    const href = await (await items[0].host()).getAttribute('href');
    const url = new URL(href, window.location.origin);
    expect(url.pathname).toBe('/report/crawlexecution');
    expect(url.searchParams.get('job_id')).toBe('job-1');
    expect(url.searchParams.get('job_execution_id')).toBe('execution-1');
  });
});
