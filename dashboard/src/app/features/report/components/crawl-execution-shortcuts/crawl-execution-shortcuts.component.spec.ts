import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {of} from 'rxjs';

import {ConfigObject, CrawlExecutionState, CrawlExecutionStatus, Kind, Meta} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  buildPlaybackUrl,
  CrawlExecutionShortcutHelpersComponent,
} from './crawl-execution-shortcuts.component';
import {AppConfig} from '../../../../app.config';
import {CrawlExecutionService} from '../../services';

describe('CrawlExecutionShortcutHelpersComponent', () => {
  let can: ReturnType<typeof vi.fn>;
  let fixture: ComponentFixture<CrawlExecutionShortcutHelpersComponent>;
  let appConfig: AppConfig;
  let getSeed: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    appConfig = {
      grpcWebUrl: 'http://localhost:8080',
      playbackBaseUrl: '',
      labelLinks: {},
    } as AppConfig;
    getSeed = vi.fn(() => of(new ConfigObject({
      meta: new Meta({name: 'https://example.com/path?key=value'}),
    })));
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionShortcutHelpersComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: AppConfig, useValue: appConfig},
        {provide: CrawlExecutionService, useValue: {getSeed}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlExecutionShortcutHelpersComponent);
  });

  function render(overrides: Partial<CrawlExecutionStatus> = {}): CrawlExecutionStatus {
    const status = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
      jobId: 'job-1',
      seedId: 'seed-1',
      ...overrides,
    });
    fixture.componentRef.setInput('crawlExecutionStatus', status);
    fixture.detectChanges();
    return status;
  }

  it('shows all permitted links in one accessible shortcut set', () => {
    render();

    expect(fixture.nativeElement.textContent).not.toContain('more_vert');
    expect(fixture.nativeElement.textContent).not.toContain('Related reports');
    expect(fixture.nativeElement.textContent).not.toContain('Related resources');
    const chipSet = fixture.nativeElement.querySelector('mat-chip-set') as HTMLElement;
    const reportLinks = fixture.nativeElement.querySelectorAll('a') as NodeListOf<HTMLAnchorElement>;
    expect(reportLinks.length).toBe(5);
    expect(fixture.nativeElement.querySelectorAll('mat-chip-set').length).toBe(1);
    expect(chipSet.getAttribute('aria-label')).toBe('Report shortcuts');
    expect([...reportLinks].every(link => link.hasAttribute('mat-chip'))).toBe(true);
    expect(reportLinks[0].textContent).toContain('Page log');
    expect(reportLinks[0].getAttribute('href')).toContain('/report/pagelog');
    expect(reportLinks[0].getAttribute('href')).toContain('execution_id=crawl-execution-1');
    expect(reportLinks[1].textContent).toContain('Crawl log');
    expect(reportLinks[1].getAttribute('href')).toContain('/report/crawllog');
    expect(reportLinks[1].getAttribute('href')).toContain('execution_id=crawl-execution-1');
  });

  it('builds a pywb URL with a UTC timestamp and validates its inputs', () => {
    expect(buildPlaybackUrl(
      '/veidemann/pywb/',
      '2026-08-07T12:34:56.789+02:00',
      'https://example.com/path?key=value',
    )).toBe('/veidemann/pywb/20260807103456/https://example.com/path?key=value');
    expect(buildPlaybackUrl('javascript:alert(1)', '2026-08-07T12:34:56Z', 'https://example.com')).toBe('');
    expect(buildPlaybackUrl('/pywb', 'not-a-date', 'https://example.com')).toBe('');
    expect(buildPlaybackUrl('/pywb', '2026-08-07T12:34:56Z', 'javascript:alert(1)')).toBe('');
  });

  it.each([
    CrawlExecutionState.FINISHED,
    CrawlExecutionState.ABORTED_TIMEOUT,
    CrawlExecutionState.ABORTED_SIZE,
    CrawlExecutionState.ABORTED_MANUAL,
    CrawlExecutionState.FAILED,
  ])('shows Playback for completed state %s', async state => {
    appConfig.playbackBaseUrl = '/veidemann/pywb';
    fixture.componentRef.setInput('showPlayback', true);
    render({state, startTime: '2026-08-07T12:34:56Z'});
    await fixture.whenStable();
    fixture.detectChanges();

    const playback = [...fixture.nativeElement.querySelectorAll('a')]
      .find((link: HTMLAnchorElement) => link.textContent.includes('Playback')) as HTMLAnchorElement;
    expect(playback.getAttribute('href'))
      .toBe('/veidemann/pywb/20260807123456/https://example.com/path?key=value');
    expect(playback.getAttribute('target')).toBe('_blank');
    expect(playback.getAttribute('rel')).toBe('noopener noreferrer');
  });

  it.each([
    CrawlExecutionState.CREATED,
    CrawlExecutionState.FETCHING,
    CrawlExecutionState.SLEEPING,
    CrawlExecutionState.DIED,
  ])('hides Playback for ineligible state %s', async state => {
    appConfig.playbackBaseUrl = '/veidemann/pywb';
    fixture.componentRef.setInput('showPlayback', true);
    render({state, startTime: '2026-08-07T12:34:56Z'});
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Playback');
    expect(getSeed).not.toHaveBeenCalled();
  });

  it('hides Playback when the base URL is not configured', async () => {
    fixture.componentRef.setInput('showPlayback', true);
    render({state: CrawlExecutionState.FINISHED, startTime: '2026-08-07T12:34:56Z'});
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Playback');
    expect(getSeed).not.toHaveBeenCalled();
  });

  it('keeps all parent-resource links visible with their existing routes', () => {
    render();

    const hrefs = [...fixture.nativeElement.querySelectorAll('a')]
      .map((link: HTMLAnchorElement) => link.getAttribute('href'));
    expect(hrefs).toEqual(expect.arrayContaining([
      '/report/jobexecution/job-execution-1',
      '/config/crawljobs/job-1',
      '/config/seed/seed-1',
    ]));
  });

  it('lets list contexts show the current Crawl Execution while suppressing log destinations', () => {
    fixture.componentRef.setInput('showPageLog', false);
    fixture.componentRef.setInput('showCrawlLog', false);
    fixture.componentRef.setInput('showCrawlExecution', true);
    render();

    const hrefs = [...fixture.nativeElement.querySelectorAll('a')]
      .map((link: HTMLAnchorElement) => link.getAttribute('href'));
    expect(hrefs).toContain('/report/crawlexecution/crawl-execution-1');
    expect(hrefs.some(href => href?.includes('/report/pagelog'))).toBe(false);
    expect(hrefs.some(href => href?.includes('/report/crawllog'))).toBe(false);
  });

  it('collapses all helper groups when no link is permitted', () => {
    can.mockReturnValue(false);
    render();

    expect(fixture.nativeElement.querySelector('mat-chip-set')).toBeNull();
  });

  it.each([
    ['pagelog', 'Page log'],
    ['crawllog', 'Crawl log'],
    ['jobexecution', 'Job execution'],
    [Kind[Kind.CRAWLJOB], 'Crawl job'],
    [Kind[Kind.SEED], 'Seed'],
  ])('applies the read permission for %s to its related link', (deniedSubject, hiddenLabel) => {
    can.mockImplementation((action: string, subject: string) =>
      action !== 'read' || subject !== deniedSubject);
    render();

    expect(fixture.nativeElement.textContent).not.toContain(hiddenLabel);
  });
});
