import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';

import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlExecutionStatusComponent', () => {
  let component: CrawlExecutionStatusComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      state: CrawlExecutionState.FINISHED,
      jobId: 'crawl-job-1',
      jobExecutionId: 'job-execution-1',
      startTime: '2025-01-01T10:00:00Z',
      endTime: '2025-01-01T10:10:00Z',
      urisCrawled: 42,
      documentsCrawled: 40,
      bytesCrawled: 1024,
    });
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the latest crawl execution as a related information card', () => {
    const card = fixture.nativeElement.querySelector('mat-card');
    expect(card.getAttribute('appearance')).toBe('filled');
    expect(card.textContent).toContain('Last crawl execution');
    expect(card.textContent).toContain('Finished');
    expect(card.textContent).toContain('42');
    expect(card.textContent).toContain('1.02 kB');
    expect(fixture.nativeElement.querySelector('mat-expansion-panel')).toBeNull();

    const hrefs = Array.from(
      fixture.nativeElement.querySelectorAll('a') as NodeListOf<HTMLAnchorElement>
    ).map(link => link.getAttribute('href'));
    expect(hrefs).toContain('/report/crawlexecution/crawl-execution-1');
    expect(hrefs).toContain('/config/crawljobs/crawl-job-1');
    expect(hrefs).toContain('/report/jobexecution/job-execution-1');
    expect(card.textContent).not.toContain('crawl-execution-1');
    expect(card.textContent).not.toContain('crawl-job-1');
    expect(card.textContent).not.toContain('job-execution-1');
    expect(card.textContent).toContain('Started');
    expect(card.textContent).toContain('Ended');
    expect(card.textContent).toContain('Duration');
    expect(card.textContent).toContain('10 min');
  });

  it('does not expose unauthorized navigation or raw identifiers', () => {
    fixture.componentRef.setInput('canReadCrawlExecution', false);
    fixture.componentRef.setInput('canReadCrawlJob', false);
    fixture.componentRef.setInput('canReadJobExecution', false);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('a').length).toBe(0);
    expect(fixture.nativeElement.textContent).not.toContain('crawl-execution-1');
  });

});
