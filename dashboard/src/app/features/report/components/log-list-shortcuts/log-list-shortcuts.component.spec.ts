import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {EMPTY, Subject, of, throwError} from 'rxjs';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, CrawlExecutionStatus, JobExecutionStatus, Meta} from '../../../../shared/models';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {LogListShortcutsComponent} from './log-list-shortcuts.component';

describe('LogListShortcutsComponent', () => {
  let fixture: ComponentFixture<LogListShortcutsComponent>;
  let can: ReturnType<typeof vi.fn>;
  let getCrawlExecution: ReturnType<typeof vi.fn>;
  let getJobExecution: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    getCrawlExecution = vi.fn(({id}: {id: string}) => of(new CrawlExecutionStatus({
      id,
      jobExecutionId: 'job-execution-1',
      jobId: 'job-1',
      seedId: 'seed-1',
    })));
    getJobExecution = vi.fn(({id}: {id: string}) => of(new JobExecutionStatus({
      id,
      jobId: 'job-1',
    })));

    await TestBed.configureTestingModule({
      imports: [LogListShortcutsComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: CrawlExecutionService, useValue: {get: getCrawlExecution}},
        {provide: JobExecutionService, useValue: {
          get: getJobExecution,
          getJob: () => of(new ConfigObject({meta: new Meta({name: 'Daily crawl'})})),
        }},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogListShortcutsComponent);
    fixture.componentRef.setInput('logKind', 'pagelog');
  });

  it('uses a crawl execution filter for the complete Page Log context without linking to either log list', async () => {
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.componentRef.setInput('jobExecutionId', 'conflicting-job-execution');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(getCrawlExecution).toHaveBeenCalledWith({id: 'crawl-execution-1', watch: false});
    expect(getJobExecution).not.toHaveBeenCalled();
    expect(fixture.nativeElement.textContent).toContain('Crawl execution');
    expect(fixture.nativeElement.textContent).toContain('Job execution');
    expect(fixture.nativeElement.textContent).toContain('Crawl job');
    expect(fixture.nativeElement.textContent).toContain('Seed');
    expect(fixture.nativeElement.textContent).not.toContain('Page log');
    expect(fixture.nativeElement.textContent).not.toContain('Crawl log');
  });

  it('adds the Page Log link for a Crawl Log execution context', async () => {
    fixture.componentRef.setInput('logKind', 'crawllog');
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.textContent).toContain('Page log');
    expect(fixture.nativeElement.textContent).not.toContain('Crawl log');
  });

  it('shows only the two unambiguous relationships for a job execution filter', async () => {
    fixture.componentRef.setInput('jobExecutionId', 'job-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(getJobExecution).toHaveBeenCalledWith({id: 'job-execution-1', watch: false});
    expect(fixture.nativeElement.textContent).toContain('Job execution');
    expect(fixture.nativeElement.textContent).toContain('Daily crawl');
    expect(fixture.nativeElement.textContent).not.toContain('Crawl executions');
    expect(fixture.nativeElement.textContent).not.toContain('Seed');
  });

  it('does not load or render context when unfiltered, unauthorized, or unavailable', async () => {
    fixture.detectChanges();
    expect(getCrawlExecution).not.toHaveBeenCalled();
    expect(getJobExecution).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();

    can.mockReturnValue(false);
    fixture.componentRef.setInput('executionId', 'unauthorized');
    fixture.detectChanges();
    expect(getCrawlExecution).not.toHaveBeenCalled();

    can.mockReturnValue(true);
    getCrawlExecution.mockReturnValue(EMPTY);
    fixture.componentRef.setInput('executionId', 'missing');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();

    getCrawlExecution.mockReturnValue(throwError(() => new Error('failed context')));
    fixture.componentRef.setInput('executionId', 'failed');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();
  });

  it('cancels stale loads and does not reload unchanged IDs', async () => {
    const first = new Subject<CrawlExecutionStatus>();
    const second = new Subject<CrawlExecutionStatus>();
    getCrawlExecution.mockImplementation(({id}: {id: string}) => id === 'first' ? first : second);

    fixture.componentRef.setInput('executionId', 'first');
    fixture.detectChanges();
    fixture.componentRef.setInput('executionId', 'second');
    fixture.detectChanges();
    first.next(new CrawlExecutionStatus({id: 'first'}));
    second.next(new CrawlExecutionStatus({id: 'second'}));
    fixture.detectChanges();
    await fixture.whenStable();

    const hrefs = [...fixture.nativeElement.querySelectorAll('a')]
      .map((link: HTMLAnchorElement) => link.getAttribute('href'));
    expect(hrefs.some(href => href?.includes('/first'))).toBe(false);
    expect(hrefs.some(href => href?.includes('/second'))).toBe(true);

    fixture.componentInstance.ngOnChanges();
    fixture.detectChanges();
    expect(getCrawlExecution).toHaveBeenCalledTimes(2);
  });
});
