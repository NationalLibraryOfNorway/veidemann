import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {JobExecutionStatusListComponent} from './job-execution-status-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  ListDataSource,
  Kind,
  Meta,
} from '../../../../shared/models';
import {JobExecutionService} from '../../services';

describe('JobExecutionStatusListComponent', () => {
  let component: JobExecutionStatusListComponent;
  let fixture: ComponentFixture<JobExecutionStatusListComponent>;
  const jobs = new Map<string, ConfigObject>();

  beforeEach(() => {
    jobs.clear();
    TestBed.configureTestingModule({
      imports: [JobExecutionStatusListComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(jobs.get(id) ?? new ConfigObject({meta: new Meta({name: 'Daily crawl'})})),
          },
        },
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the job name as plain row content and keeps row navigation', async () => {
    const row = new JobExecutionStatus({id: 'execution-1', jobId: 'job-1'});
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    const clicked: JobExecutionStatus[] = [];
    component.rowClick.subscribe(item => clicked.push(item));

    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const itemRow = fixture.nativeElement.querySelector('.item-row') as HTMLElement;
    const jobCell = itemRow.querySelector('td') as HTMLElement;
    expect(jobCell.textContent).toContain('Daily crawl');
    expect(jobCell.querySelector('a')).toBeNull();

    itemRow.click();
    itemRow.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(clicked).toEqual([row, row]);
  });

  it('renders execution metrics in the shared column order and appends duration', async () => {
    const row = new JobExecutionStatus({
      id: 'execution-1',
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
      startTime: '2026-08-11T10:00:00.000Z',
      endTime: '2026-08-11T11:02:00.000Z',
      documentsCrawled: 1234,
      bytesCrawled: 1500,
    });
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(component.displayedColumns).toEqual([
      'jobId',
      'state',
      'queueSize',
      'documentsCrawled',
      'bytesCrawled',
      'startTime',
      'endTime',
      'duration',
      'action',
    ]);
    const headers = [...fixture.nativeElement.querySelectorAll('th')]
      .map((header: HTMLElement) => header.textContent.trim());
    expect(headers).toEqual([
      'Job', 'State', 'Queue', 'Documents', 'Bytes', 'Start', 'End', 'Duration', '',
    ]);
    expect(fixture.nativeElement.querySelector('.mat-column-desiredState')).toBeNull();
    expect(fixture.nativeElement.querySelector('td.mat-column-documentsCrawled').textContent.trim())
      .toBe('1,234');
    expect(fixture.nativeElement.querySelector('td.mat-column-bytesCrawled').textContent.trim())
      .toBe('1.5 kB');
    expect(fixture.nativeElement.querySelector('td.mat-column-endTime').textContent.trim())
      .not.toContain('Aborted manually');
    expect(fixture.nativeElement.querySelector('td.mat-column-duration').textContent.trim())
      .toBe('1hours:2min');
  });

  describe('maximum duration', () => {
    const job = (maxDurationS: number) => new ConfigObject({
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Limited crawl'}),
      crawlJob: new CrawlJob({limits: {maxDurationS, maxBytes: 0}}),
    });

    async function render(...rows: JobExecutionStatus[]): Promise<HTMLElement[]> {
      fixture.componentRef.setInput('dataSource', ListDataSource.fromQuery({
        query$: of('query'),
        load: () => of(...rows),
        destroyRef: fixture.componentRef.injector.get(DestroyRef),
      }));
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();
      return [...fixture.nativeElement.querySelectorAll('td.mat-column-duration')] as HTMLElement[];
    }

    it('does not mark ended executions, whatever their duration', async () => {
      jobs.set('limited', job(3600));
      const [ended] = await render(new JobExecutionStatus({
        id: 'ended', jobId: 'limited', state: JobExecutionState.FINISHED,
        startTime: '2026-08-11T10:00:00.000Z', endTime: '2026-08-11T22:30:00.000Z',
      }));

      expect(ended.querySelector('.over-max-duration')).toBeNull();
      expect(ended.querySelector('.visually-hidden')).toBeNull();
      expect(ended.textContent.trim()).toBe('12hours:30min');
    });

    it('shows a badge with the overrun behind the duration of running executions', async () => {
      jobs.set('limited', job(3600));
      const [running] = await render(new JobExecutionStatus({
        id: 'running', jobId: 'limited', state: JobExecutionState.RUNNING,
        startTime: new Date(Date.now() - 5 * 3600_000).toISOString(),
      }));

      const badge = running.querySelector('.over-max-duration') as HTMLElement;
      expect(badge).not.toBeNull();
      expect(badge.textContent).toContain('+4 h');
      expect(badge.querySelector('mat-icon')?.getAttribute('aria-hidden')).toBe('true');
      expect(running.textContent.indexOf('5hours')).toBeLessThan(running.textContent.indexOf('+4 h'));
    });

    it('does not mark executions of jobs without a limit or without a start', async () => {
      jobs.set('unlimited', job(0));
      const cells = await render(
        new JobExecutionStatus({
          id: 'unlimited', jobId: 'unlimited', state: JobExecutionState.FINISHED,
          startTime: '2026-08-11T10:00:00.000Z', endTime: '2026-08-14T10:00:00.000Z',
        }),
        new JobExecutionStatus({id: 'no-start', jobId: 'unlimited', state: JobExecutionState.RUNNING}),
      );

      expect(cells.map(cell => cell.querySelector('.over-max-duration'))).toEqual([null, null]);
    });
  });

  it('uses desired state or missing-value text when an end timestamp is absent', async () => {
    const rows = [
      new JobExecutionStatus({
        id: 'requested',
        state: JobExecutionState.RUNNING,
        desiredState: JobExecutionState.ABORTED_MANUAL,
      }),
      new JobExecutionStatus({
        id: 'active',
        state: JobExecutionState.RUNNING,
      }),
      new JobExecutionStatus({
        id: 'terminal',
        state: JobExecutionState.FAILED,
      }),
      new JobExecutionStatus({
        id: 'undefined',
        state: JobExecutionState.UNDEFINED,
      }),
    ];
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(...rows),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const endCells = [...fixture.nativeElement.querySelectorAll('td.mat-column-endTime')]
      .map((cell: HTMLElement) => cell.textContent.trim());
    expect(endCells).toEqual(['Aborted manually', '', 'Not available', 'Not available']);
  });
});
