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

  it('renders only the external requested URI action without an overflow menu', async () => {
    const row = new CrawlLog({warcId: 'warc-1', requestedUri: 'https://example.org/request'});
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const actionCell = fixture.nativeElement.querySelector('td.mat-column-action') as HTMLElement;
    const links = actionCell.querySelectorAll('a') as NodeListOf<HTMLAnchorElement>;
    expect(component.displayedColumns).toEqual([
      'requestedUri', 'timestamp', 'statusCode', 'discoveryPath', 'contentType', 'action',
    ]);
    expect(links.length).toBe(1);
    expect(links[0].href).toBe('https://example.org/request');
    expect(links[0].target).toBe('_blank');
    expect(actionCell.querySelector('[aria-label="More actions"]')).toBeNull();
    expect(actionCell.querySelector('mat-menu')).toBeNull();
  });
});
