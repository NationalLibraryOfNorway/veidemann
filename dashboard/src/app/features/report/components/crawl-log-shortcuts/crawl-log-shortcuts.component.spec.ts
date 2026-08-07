import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {CrawlLogShortcutsComponent} from './crawl-log-shortcuts.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {SnackBarService} from '../../../../core';
import {CrawlLog} from '../../../../shared/models';

describe('CrawlLogShortcutsComponent', () => {
  let component: CrawlLogShortcutsComponent;
  let fixture: ComponentFixture<CrawlLogShortcutsComponent>;
  let can: ReturnType<typeof vi.fn>;
  let copy: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    copy = vi.fn(() => true);
    await TestBed.configureTestingModule({
      imports: [CrawlLogShortcutsComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: Clipboard, useValue: {copy}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn(), openError: vi.fn()}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogShortcutsComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('crawlLog', new CrawlLog());
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the copyable log ID and linked execution IDs in one chip set', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      warcId: 'warc-1',
      executionId: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
    }));
    fixture.detectChanges();

    const chipSet = fixture.nativeElement.querySelector('mat-chip-set') as HTMLElement;
    const links = fixture.nativeElement.querySelectorAll('a[mat-chip]') as NodeListOf<HTMLAnchorElement>;
    expect(chipSet.getAttribute('aria-label')).toBe('Crawl log identifiers');
    expect(fixture.nativeElement.textContent).toContain('ID: warc-1');
    expect([...links].map(link => link.getAttribute('href'))).toEqual([
      '/report/crawlexecution/crawl-execution-1',
      '/report/jobexecution/job-execution-1',
    ]);

    (fixture.nativeElement.querySelector('[aria-label="Copy crawl log ID"]') as HTMLElement).click();
    expect(copy).toHaveBeenCalledWith('warc-1');
  });

  it('falls back to plain execution ID chips without read permission', () => {
    can.mockReturnValue(false);
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      warcId: 'warc-1',
      executionId: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('a[mat-chip]').length).toBe(0);
    expect(fixture.nativeElement.textContent).toContain('Crawl execution ID: crawl-execution-1');
    expect(fixture.nativeElement.textContent).toContain('Job execution ID: job-execution-1');
  });
});
