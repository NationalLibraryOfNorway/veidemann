import {formatDate} from '@angular/common';
import {LOCALE_ID} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {EMPTY, Subject, of, throwError} from 'rxjs';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, CrawlExecutionState, CrawlExecutionStatus, Kind, Meta} from '../../../../shared/models';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {LogListShortcutsComponent} from './log-list-shortcuts.component';

describe('LogListShortcutsComponent', () => {
  let fixture: ComponentFixture<LogListShortcutsComponent>;
  let can: ReturnType<typeof vi.fn>;
  let getCrawlExecution: ReturnType<typeof vi.fn>;
  let getSeed: ReturnType<typeof vi.fn>;
  let getJob: ReturnType<typeof vi.fn>;
  let locale: string;

  function localizedMedium(value: string): string {
    return formatDate(value, 'medium', locale).replace(/\s+/g, ' ');
  }

  beforeEach(async () => {
    can = vi.fn(() => true);
    getCrawlExecution = vi.fn(({id}: {id: string}) => of(new CrawlExecutionStatus({
      id,
      jobExecutionId: 'job-execution-1',
      jobId: 'job-1',
      seedId: 'seed-1',
      state: CrawlExecutionState.FINISHED,
      startTime: '2026-08-13T22:49:12.000Z',
      endTime: '2026-08-13T22:49:24.000Z',
    })));
    getSeed = vi.fn((id: string) => of(new ConfigObject({
      id,
      kind: Kind.SEED,
      meta: new Meta({name: 'Example seed'}),
    })));
    getJob = vi.fn((id: string) => of(new ConfigObject({
      id,
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Unscheduled'}),
    })));

    await TestBed.configureTestingModule({
      imports: [LogListShortcutsComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: CrawlExecutionService, useValue: {get: getCrawlExecution, getSeed}},
        {provide: JobExecutionService, useValue: {getJob}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogListShortcutsComponent);
    fixture.componentRef.setInput('logKind', 'pagelog');
    locale = TestBed.inject(LOCALE_ID);
  });

  it('renders the filtered execution metadata beside the Page Log menu', async () => {
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(getCrawlExecution).toHaveBeenCalledOnce();
    expect(getCrawlExecution).toHaveBeenCalledWith({id: 'crawl-execution-1', watch: false});
    expect(getSeed).toHaveBeenCalledOnce();
    expect(getSeed).toHaveBeenCalledWith('seed-1');
    expect(getJob).toHaveBeenCalledOnce();
    expect(getJob).toHaveBeenCalledWith('job-1');

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    const labels = [...metadata.querySelectorAll('dt')]
      .map((term: HTMLElement) => term.textContent.trim());
    const values = [...metadata.querySelectorAll('dd')]
      .map((value: HTMLElement) => value.textContent.replace(/\s+/g, ' ').trim());
    expect(labels).toEqual(['Started', 'Finished', 'Crawl job', 'Seed']);
    expect(values).toEqual([
      localizedMedium('2026-08-13T22:49:12.000Z'),
      localizedMedium('2026-08-13T22:49:24.000Z'),
      'Unscheduled',
      'Example seed',
    ]);
    expect(metadata.querySelector('.execution-lifecycle')?.classList)
      .toContain('execution-lifecycle--metrics');
    expect(fixture.nativeElement.querySelector('mat-chip')).toBeNull();
    expect(getComputedStyle(fixture.nativeElement).display).toBe('contents');
    expect(getComputedStyle(metadata).gridRowStart).toBe('1');

    const menu = await openMenu('Page log actions');
    expect(getComputedStyle(fixture.nativeElement.querySelector('app-detail-overflow')).gridRowStart).toBe('1');
    expect(menu.textContent).toContain('Crawl execution');
    expect(menu.textContent).toContain('Job execution');
    expect(menu.textContent).toContain('Crawl job');
    expect(menu.textContent).toContain('Seed');
    expect(menu.textContent).not.toContain('Copy ID');
    expect(menu.textContent).not.toContain('Page log');
    expect(menu.textContent).not.toContain('Crawl log');
    expect(menuLinks(menu)).toEqual([
      '/report/crawlexecution/crawl-execution-1',
      '/report/jobexecution/job-execution-1',
      '/config/crawljobs/job-1',
      '/config/seed/seed-1',
    ]);
  });

  it('adds the Page Log action for a Crawl Log execution context', async () => {
    fixture.componentRef.setInput('logKind', 'crawllog');
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    const menu = await openMenu('Crawl log actions');
    expect(menu.textContent).toContain('Page log');
    expect(menu.textContent).not.toContain('Crawl log');
    expect(menu.textContent).not.toContain('Copy ID');
    expect(menuLinks(menu)[0]).toBe('/report/pagelog?execution_id=crawl-execution-1');
  });

  it('falls back to the crawl-job ID and deleted-seed label', async () => {
    getJob.mockReturnValue(of(new ConfigObject({meta: new Meta()})));
    getSeed.mockReturnValue(of(null));
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const values = [...fixture.nativeElement.querySelectorAll('app-execution-metadata dd')]
      .map((value: HTMLElement) => value.textContent.trim());
    expect(values[2]).toBe('job-1');
    expect(values[3]).toBe('Deleted seed');
  });

  it('omits context when unfiltered, unauthorized, or unavailable', async () => {
    fixture.detectChanges();
    expect(getCrawlExecution).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('app-execution-metadata')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();

    can.mockReturnValue(false);
    fixture.componentRef.setInput('executionId', 'unauthorized');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(getCrawlExecution).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('app-execution-metadata')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();

    can.mockReturnValue(true);
    getCrawlExecution.mockReturnValue(EMPTY);
    fixture.componentRef.setInput('executionId', 'missing');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('app-execution-metadata')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();

    getCrawlExecution.mockReturnValue(throwError(() => new Error('failed context')));
    fixture.componentRef.setInput('executionId', 'failed');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('app-execution-metadata')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();
  });

  it('cancels stale context loads and does not reload unchanged IDs', async () => {
    const first = new Subject<CrawlExecutionStatus>();
    const second = new Subject<CrawlExecutionStatus>();
    getCrawlExecution.mockImplementation(({id}: {id: string}) => id === 'first' ? first : second);
    getSeed.mockImplementation((id: string) => of(new ConfigObject({
      id,
      kind: Kind.SEED,
      meta: new Meta({name: id === 'second-seed' ? 'Second seed' : 'First seed'}),
    })));
    getJob.mockImplementation((id: string) => of(new ConfigObject({
      id,
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: id === 'second-job' ? 'Second job' : 'First job'}),
    })));

    fixture.componentRef.setInput('executionId', 'first');
    fixture.detectChanges();
    fixture.componentRef.setInput('executionId', 'second');
    fixture.detectChanges();
    first.next(new CrawlExecutionStatus({id: 'first', seedId: 'first-seed', jobId: 'first-job'}));
    second.next(new CrawlExecutionStatus({id: 'second', seedId: 'second-seed', jobId: 'second-job'}));
    fixture.detectChanges();
    await fixture.whenStable();

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain('Second seed');
    expect(metadata.textContent).toContain('Second job');
    expect(metadata.textContent).not.toContain('First seed');
    expect(metadata.textContent).not.toContain('First job');
    const menu = await openMenu('Page log actions');
    const hrefs = menuLinks(menu);
    expect(hrefs.some(href => href?.includes('/first'))).toBe(false);
    expect(hrefs.some(href => href?.includes('/second'))).toBe(true);

    fixture.componentInstance.ngOnChanges();
    fixture.detectChanges();
    expect(getCrawlExecution).toHaveBeenCalledTimes(2);
  });

  async function openMenu(label: string): Promise<HTMLElement> {
    const trigger = fixture.nativeElement.querySelector(`button[aria-label="${label}"]`) as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    expect(trigger.querySelector('mat-icon')?.textContent).toContain('more_vert');
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve));
    return document.querySelector('.mat-mdc-menu-panel') as HTMLElement;
  }

  function menuLinks(menu: HTMLElement): (string | null)[] {
    return [...menu.querySelectorAll('a')]
      .map((link: HTMLAnchorElement) => link.getAttribute('href'));
  }
});
