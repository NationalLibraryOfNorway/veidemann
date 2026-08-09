import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
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

  it('offers unique normalized content types from loaded rows and filters by type', async () => {
    fixture.detectChanges();
    const chips = [...fixture.nativeElement.querySelectorAll('.report-filter-toolbar > .facet-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());
    expect(chips).toEqual(['image/png', 'text/html', 'text/plain']);
    expect(component.dataSource.snapshot.map(row => row.id))
      .toEqual(['html', 'html-duplicate', 'plain', 'image', 'missing']);

    component.onContentTypeFilterChange(['text/html']);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['html', 'html-duplicate']);
    expect(component.loadedContentTypes()).toEqual(['image/png', 'text/html', 'text/plain']);

    component.onContentTypeFilterChange(['text/plain', 'image/png']);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['plain', 'image']);

    component.onContentTypeFilterChange([]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id))
      .toEqual(['html', 'html-duplicate', 'plain', 'image', 'missing']);
  });

  it('shows status families, exact codes, and methods from loaded logs only', () => {
    fixture.detectChanges();
    const statusChips = [...fixture.nativeElement.querySelectorAll('app-http-status-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());
    const methodChips = [...fixture.nativeElement.querySelectorAll('.method-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());

    expect(component.loadedStatusFamilies()).toEqual([2, 3, 4, 5]);
    expect(component.loadedStatusCodes()).toEqual([200, 302, 404, 500]);
    expect(statusChips).toEqual(['2xx', '3xx', '4xx', '5xx', '200', '302', '404', '500']);
    expect(component.loadedMethods()).toEqual(['GET', 'HEAD', 'POST']);
    expect(methodChips).toEqual(['GET', 'HEAD', 'POST']);
  });

  it('combines MIME type, method, and HTTP status filters without loading more', async () => {
    component.onStatusFamilyFilterChange([2, 3]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['html', 'plain']);

    component.onStatusFamilyFilterChange([]);
    component.onExactStatusFilterChange([404]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['html-duplicate']);

    component.onExactStatusFilterChange([]);
    component.onStatusFamilyFilterChange([2, 3]);
    component.onMethodFilterChange(['HEAD']);
    await fixture.whenStable();
    expect(component.dataSource.snapshot.map(row => row.id)).toEqual(['plain']);

    component.onContentTypeFilterChange(['text/plain']);
    component.onExactStatusFilterChange([]);
    component.onStatusFamilyFilterChange([4]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot).toEqual([]);
    expect(component.hasClientFilters()).toBe(true);

    component.onContentTypeFilterChange([]);
    component.onMethodFilterChange([]);
    component.onStatusFamilyFilterChange([]);
    component.onExactStatusFilterChange([]);
    await fixture.whenStable();
    expect(component.dataSource.snapshot).toHaveLength(5);
    expect(component.hasClientFilters()).toBe(false);
  });
});
