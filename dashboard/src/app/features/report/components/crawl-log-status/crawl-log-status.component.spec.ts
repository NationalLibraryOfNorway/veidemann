import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogStatusComponent} from './crawl-log-status.component';
import {CrawlLog} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';

describe('CrawlLogStatusComponent', () => {
  let component: CrawlLogStatusComponent;
  let fixture: ComponentFixture<CrawlLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CrawlLogStatusComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: ActivatedRoute, useValue: {}}
      ],

    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogStatusComponent);
    component = fixture.componentInstance;
    component.crawlLog = new CrawlLog();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('uses a helper-ready card header and no longer renders the identifiers table', () => {
    component.crawlLog = new CrawlLog({
      warcId: 'warc-1',
      executionId: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
    });
    component.ngOnInit();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-header.card-header-with-helpers')).not.toBeNull();
    expect(fixture.nativeElement.querySelectorAll('table').length).toBe(3);
    expect(fixture.nativeElement.textContent).not.toContain('Crawl execution id');
    expect(fixture.nativeElement.textContent).not.toContain('Job execution id');
  });
});
