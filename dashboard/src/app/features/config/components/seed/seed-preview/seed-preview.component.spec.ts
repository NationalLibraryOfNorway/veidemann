import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {NEVER, of, throwError} from 'rxjs';

import {ReportApiService} from '../../../../../core/api/report-api.service';
import {AuthService} from '../../../../../core/auth';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {
  Annotation,
  ConfigObject,
  ConfigRef,
  CrawlExecutionState,
  CrawlExecutionStatus,
  Kind,
  Label,
  Meta,
  Seed,
} from '../../../../../shared/models';
import {ConfigService} from '../../../../../shared/services';
import {SeedPreviewComponent} from './seed-preview.component';

describe('SeedPreviewComponent', () => {
  let fixture: ComponentFixture<SeedPreviewComponent>;
  const getLastSeedStatus = vi.fn();
  const get = vi.fn();

  const seed = new ConfigObject({
    id: 'seed-1',
    kind: Kind.SEED,
    meta: new Meta({
      name: 'https://example.test/',
      created: '2024-06-01T08:00:00Z',
      createdBy: 'creator',
      labelList: [new Label({key: 'topic', value: 'news'})],
      annotationList: [new Annotation({key: 'scope_maxDepth', value: '3'})],
    }),
    seed: new Seed({
      entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-1'}),
      jobRefList: [
        new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}),
        new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-unknown'}),
      ],
    }),
  });
  const crawlJobs = [
    new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily crawl'})}),
  ];

  const factText = (label: string): string | undefined =>
    ([...fixture.nativeElement.querySelectorAll('.preview-facts > div')] as HTMLElement[])
      .find(fact => fact.querySelector('dt')?.textContent.trim() === label)
      ?.querySelector('dd')?.textContent.replace(/\s+/g, ' ').trim();

  beforeEach(async () => {
    getLastSeedStatus.mockReset();
    get.mockReset();
    await TestBed.configureTestingModule({
      imports: [SeedPreviewComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: ReportApiService, useValue: {getLastSeedStatus}},
        {provide: ConfigService, useValue: {get}},
        {provide: AuthService, useValue: {canRead: () => true}},
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(SeedPreviewComponent);
  });

  const render = () => {
    fixture.componentRef.setInput('seed', seed);
    fixture.componentRef.setInput('crawlJobs', crawlJobs);
    fixture.detectChanges();
  };

  it('lists recent crawl executions with job, start, end and status, plus seed metadata', () => {
    getLastSeedStatus.mockReturnValue(of(
      new CrawlExecutionStatus({
        id: 'crawl-execution-2',
        state: CrawlExecutionState.FETCHING,
        jobId: 'job-1',
        startTime: '2025-01-02T10:00:00Z',
      }),
      new CrawlExecutionStatus({
        id: 'crawl-execution-1',
        state: CrawlExecutionState.ABORTED_MANUAL,
        jobId: 'job-unknown',
        startTime: '2025-01-01T10:00:00Z',
        endTime: '2025-01-01T10:10:00Z',
      }),
    ));
    get.mockReturnValue(of(new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({name: 'Example entity'}),
    })));
    render();

    expect(getLastSeedStatus).toHaveBeenCalledWith('seed-1', 3);
    expect(get).toHaveBeenCalledWith(seed.seed.entityRef, {suppressNotFound: true});
    const rows = ([...fixture.nativeElement.querySelectorAll('.crawl-execution-table tr[mat-row]')] as HTMLElement[])
      .map(row => ([...row.querySelectorAll('td')] as HTMLElement[]).map(cell => cell.textContent.replace(/\s+/g, ' ').trim()));
    expect(rows.length).toBe(2);
    expect(rows[0][0]).toBe('Daily crawl');
    expect(rows[0][2]).toBe('–');
    expect(rows[0][3]).toContain('Fetching');
    expect(rows[1][0]).toBe('job-unknown');
    expect(rows[1][2]).not.toBe('–');
    expect(rows[1][3]).toContain('Aborted manually');
    const stateLink = fixture.nativeElement.querySelector('.execution-state a') as HTMLElement;
    expect(stateLink.getAttribute('href')).toBe('/report/crawlexecution/crawl-execution-2');
    expect(fixture.nativeElement.querySelectorAll('.execution-state-error').length).toBe(1);

    expect(factText('Entity')).toBe('Example entity');
    expect(factText('Crawl jobs')).toBe('Daily crawl, job-unknown');
    expect(factText('ID')).toContain('seed-1');
    expect(factText('Labels')).toContain('topic');
    expect(factText('Script annotations')).toContain('scope_maxDepth');
    expect(factText('Created')).toContain('creator');
    expect(fixture.nativeElement.querySelector('app-effective-script-annotations')).toBeNull();
    const columns = [...fixture.nativeElement.querySelector('.seed-preview').children] as HTMLElement[];
    expect(columns.map(column => column.className))
      .toEqual(['preview-section seed-metadata', 'preview-facts seed-relations', 'preview-section recent-crawls']);
    const title = fixture.nativeElement.querySelector('.seed-title a') as HTMLElement;
    expect(title.textContent.trim()).toBe('https://example.test/');
    expect(title.getAttribute('href')).toBe('/config/seed/seed-1');
    const columnFacts = (column: HTMLElement) => ([...column.querySelectorAll('dt')] as HTMLElement[])
      .map(term => term.textContent.trim());
    expect(columnFacts(columns[0])).toEqual(['ID', 'Created', 'Entity']);
    expect(columnFacts(columns[1])).toEqual(['Crawl jobs', 'Labels', 'Script annotations']);
  });

  it('shows an empty state and falls back to the entity ID when nothing is available', () => {
    getLastSeedStatus.mockReturnValue(of(null));
    get.mockReturnValue(throwError(() => new Error('unavailable')));
    render();

    expect(fixture.nativeElement.querySelector('.crawl-execution-table')).toBeNull();
    expect(fixture.nativeElement.querySelector('.empty-state').textContent).toContain('not been crawled');
    expect(factText('Entity')).toBe('entity-1');
  });

  it('shows a progress bar while crawl executions are loading', () => {
    getLastSeedStatus.mockReturnValue(NEVER);
    get.mockReturnValue(of(null));
    render();

    expect(fixture.nativeElement.querySelector('mat-progress-bar')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.empty-state')).toBeNull();
  });

  it('shows an error instead of the empty state when crawl executions fail to load', () => {
    getLastSeedStatus.mockReturnValue(throwError(() => new Error('unavailable')));
    get.mockReturnValue(of(null));
    render();

    expect(fixture.nativeElement.querySelector('mat-progress-bar')).toBeNull();
    expect(fixture.nativeElement.querySelector('.empty-state-error').textContent).toContain('Could not load');
  });
});
