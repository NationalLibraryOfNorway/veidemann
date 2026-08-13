import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {EMPTY, Subject, of, throwError} from 'rxjs';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, CrawlExecutionStatus, Kind, Meta} from '../../../../shared/models';
import {CrawlExecutionService} from '../../services';
import {LogListShortcutsComponent} from './log-list-shortcuts.component';

describe('LogListShortcutsComponent', () => {
  let fixture: ComponentFixture<LogListShortcutsComponent>;
  let can: ReturnType<typeof vi.fn>;
  let getCrawlExecution: ReturnType<typeof vi.fn>;
  let getSeed: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    getCrawlExecution = vi.fn(({id}: {id: string}) => of(new CrawlExecutionStatus({
      id,
      jobExecutionId: 'job-execution-1',
      jobId: 'job-1',
      seedId: 'seed-1',
    })));
    getSeed = vi.fn((id: string) => of(new ConfigObject({
      id,
      kind: Kind.SEED,
      meta: new Meta({name: 'Example seed'}),
    })));

    await TestBed.configureTestingModule({
      imports: [LogListShortcutsComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: CrawlExecutionService, useValue: {get: getCrawlExecution, getSeed}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogListShortcutsComponent);
    fixture.componentRef.setInput('logKind', 'pagelog');
  });

  it('reuses one Crawl Execution context to resolve a prefix-free Seed chip and Page Log menu', async () => {
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(getCrawlExecution).toHaveBeenCalledOnce();
    expect(getCrawlExecution).toHaveBeenCalledWith({id: 'crawl-execution-1', watch: false});
    expect(getSeed).toHaveBeenCalledOnce();
    expect(getSeed).toHaveBeenCalledWith('seed-1');

    const chip = fixture.nativeElement.querySelector('.log-context-filter mat-chip') as HTMLElement;
    expect(chip.textContent).toContain('Example seed');
    expect(chip.textContent).not.toContain('Seed:');
    expect(chip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('link');
    expect(chip.querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove crawl execution crawl-execution-1 filter');

    const menu = await openMenu('Page log actions');
    expect(menu.textContent).toContain('Crawl execution');
    expect(menu.textContent).toContain('Job execution');
    expect(menu.textContent).toContain('Crawl job');
    expect(menu.textContent).toContain('Seed');
    expect(menu.textContent).toContain('Copy ID');
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
    expect(menu.textContent).toContain('Copy ID');
    expect(menuLinks(menu)[0]).toBe('/report/pagelog?execution_id=crawl-execution-1');
  });

  it('emits removal for the exact execution filter', async () => {
    let removed = false;
    fixture.componentInstance.removeExecutionFilter.subscribe(() => removed = true);
    fixture.componentRef.setInput('executionId', 'crawl-execution-1');
    fixture.detectChanges();
    await fixture.whenStable();

    (fixture.nativeElement.querySelector('button[matChipRemove]') as HTMLButtonElement).click();

    expect(removed).toBe(true);
  });

  it('keeps an ID fallback chip but omits the menu when context is unauthorized or unavailable', async () => {
    fixture.detectChanges();
    expect(getCrawlExecution).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('mat-chip')).toBeNull();

    can.mockReturnValue(false);
    fixture.componentRef.setInput('executionId', 'unauthorized');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(getCrawlExecution).not.toHaveBeenCalled();
    expect(fallbackChipText()).toContain('Crawl execution: unauthorized');
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();

    can.mockReturnValue(true);
    getCrawlExecution.mockReturnValue(EMPTY);
    fixture.componentRef.setInput('executionId', 'missing');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fallbackChipText()).toContain('Crawl execution: missing');
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();

    getCrawlExecution.mockReturnValue(throwError(() => new Error('failed context')));
    fixture.componentRef.setInput('executionId', 'failed');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fallbackChipText()).toContain('Crawl execution: failed');
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

    fixture.componentRef.setInput('executionId', 'first');
    fixture.detectChanges();
    fixture.componentRef.setInput('executionId', 'second');
    fixture.detectChanges();
    first.next(new CrawlExecutionStatus({id: 'first', seedId: 'first-seed'}));
    second.next(new CrawlExecutionStatus({id: 'second', seedId: 'second-seed'}));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('mat-chip')?.textContent).toContain('Second seed');
    expect(fixture.nativeElement.querySelector('mat-chip')?.textContent).not.toContain('First seed');
    const menu = await openMenu('Page log actions');
    const hrefs = menuLinks(menu);
    expect(hrefs.some(href => href?.includes('/first'))).toBe(false);
    expect(hrefs.some(href => href?.includes('/second'))).toBe(true);

    fixture.componentInstance.ngOnChanges();
    fixture.detectChanges();
    expect(getCrawlExecution).toHaveBeenCalledTimes(2);
  });

  function fallbackChipText(): string {
    const chip = fixture.nativeElement.querySelector('.log-context-filter mat-chip') as HTMLElement;
    expect(chip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('hdr_weak');
    return chip.textContent;
  }

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
