import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {of} from 'rxjs';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlLog} from '../../../../shared/models';
import {CrawlLogService} from '../../services';
import {CrawlLogComponent} from './crawl-log.component';

describe('CrawlLogComponent', () => {
  let fixture: ComponentFixture<CrawlLogComponent>;
  let component: CrawlLogComponent;

  beforeEach(async () => {
    const rows = [
      new CrawlLog({warcId: 'html', method: 'GET', contentType: 'Text/HTML; charset=UTF-8', statusCode: 200}),
      new CrawlLog({warcId: 'html-duplicate', method: 'get', contentType: 'text/html; charset=ISO-8859-1', statusCode: 404}),
      new CrawlLog({warcId: 'plain', method: 'HEAD', contentType: 'text/plain', statusCode: 302}),
      new CrawlLog({warcId: 'image', method: 'POST', contentType: 'image/png', statusCode: 500}),
      new CrawlLog({warcId: 'missing', contentType: '  ', statusCode: 0}),
    ];
    await TestBed.configureTestingModule({
      imports: [CrawlLogComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: CrawlLogService,
          useValue: {
            loading$: of(false),
            search: () => of(...rows),
          },
        },
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlLogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('renders the list without a filter toolbar or execution actions menu', () => {
    fixture.detectChanges();
    expect(component.dataSource.snapshot).toHaveLength(5);
    expect(fixture.nativeElement.querySelector('app-crawl-log-list')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.crawl-log-facets')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.report-filter-toolbar')).toBeNull();
    const executionQuery = fixture.nativeElement.querySelector('app-crawl-log-query') as HTMLElement;
    expect(executionQuery).not.toBeNull();
    expect((executionQuery.querySelector('mat-form-field') as HTMLElement).hidden).toBe(true);
    expect(fixture.nativeElement.querySelector('app-log-list-shortcuts')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();
  });

  it('keeps execution-ID query changes wired to the route', () => {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);

    component.onQueryChange({...component.query(), executionId: 'execution-2'});

    expect(navigate).toHaveBeenCalledWith([], expect.objectContaining({
      queryParamsHandling: 'merge',
      queryParams: expect.objectContaining({
        p: null,
        s: null,
        execution_id: 'execution-2',
        job_execution_id: null,
      }),
    }));
  });

  it('preserves the visible MIME type, HTTP status, and method facets', () => {
    fixture.detectChanges();
    const contentTypeChips = [...fixture.nativeElement.querySelectorAll(
      '.crawl-log-facets > .facet-filter mat-chip-option'
    )].map((chip: Element) => chip.textContent?.trim());
    const statusChips = [...fixture.nativeElement.querySelectorAll('app-http-status-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());
    const methodChips = [...fixture.nativeElement.querySelectorAll('.method-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());

    expect(contentTypeChips).toEqual(['image/png', 'text/html', 'text/plain']);
    expect(statusChips).toEqual(['2xx', '3xx', '4xx', '5xx', '200', '302', '404', '500']);
    expect(methodChips).toEqual(['GET', 'HEAD', 'POST']);
  });

  it('combines the preserved client facets without loading more', async () => {
    component.onStatusFamilyFilterChange([2, 3]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['html', 'plain']);

    component.onStatusFamilyFilterChange([]);
    component.onExactStatusFilterChange([404]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['html-duplicate']);

    component.onExactStatusFilterChange([]);
    component.onMethodFilterChange(['HEAD']);
    component.onContentTypeFilterChange(['text/plain']);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['plain']);
    expect(component.hasClientFilters()).toBe(true);

    component.onContentTypeFilterChange([]);
    component.onMethodFilterChange([]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot).toHaveLength(5);
    expect(component.hasClientFilters()).toBe(false);
  });
});
