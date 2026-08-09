import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {CrawlLogListComponent} from './crawl-log-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlLog, ListDataSource} from '../../../../shared/models';

describe('CrawlLogListComponent', () => {
  let component: CrawlLogListComponent;
  let fixture: ComponentFixture<CrawlLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlLogListComponent],
      providers: [
        ...provideCoreTesting,
      ],
      declarations: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the requested URI as an external link without a separate action column', async () => {
    const row = new CrawlLog({warcId: 'warc-1', method: 'GET', requestedUri: 'https://example.org/request'});
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const links = fixture.nativeElement.querySelectorAll('td.mat-column-requestedUri a') as NodeListOf<HTMLAnchorElement>;
    expect(component.displayedColumns).toEqual([
      'method', 'requestedUri', 'statusCode', 'contentType', 'discoveryPath', 'timestamp',
    ]);
    expect(fixture.nativeElement.querySelector('td.mat-column-method')?.textContent).toContain('GET');
    expect(links.length).toBe(1);
    expect(links[0].href).toBe('https://example.org/request');
    expect(links[0].target).toBe('_blank');
    expect(fixture.nativeElement.querySelector('.mat-column-action')).toBeNull();
    expect(fixture.nativeElement.querySelector('[aria-label="Open requested URI in a new tab"]')).toBeNull();
  });
});
