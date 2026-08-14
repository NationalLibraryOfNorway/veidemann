import {Component, DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatMenuItem} from '@angular/material/menu';
import {By} from '@angular/platform-browser';
import {provideRouter, RouterLink} from '@angular/router';
import {of} from 'rxjs';

import {CrawlExecutionStatusListComponent} from './crawl-execution-status-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  CrawlExecutionState,
  CrawlExecutionStatus,
  ListDataSource,
  Meta
} from '../../../../shared/models';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {ActionDirective} from '../../../../shared/directives';

@Component({
  template: `
    <app-crawl-execution-status-list
      [dataSource]="dataSource"
      [hasOverflowActions]="hasOverflowActions"
      (rowClick)="clicked.push($event)">
      <ng-container *appAction="let row">
        <a class="page-log-action" mat-menu-item [routerLink]="['/report/pagelog']"
          [queryParams]="{execution_id: row.id}"><mat-icon>art_track</mat-icon></a>
        <a class="crawl-log-action" mat-menu-item [routerLink]="['/report/crawllog']"
          [queryParams]="{execution_id: row.id}"><mat-icon>event_note</mat-icon></a>
        <button mat-menu-item type="button">Abort {{row.id}}</button>
      </ng-container>
    </app-crawl-execution-status-list>
  `,
  imports: [
    ActionDirective,
    CrawlExecutionStatusListComponent,
    MatButtonModule,
    MatIcon,
    MatMenuItem,
    RouterLink,
  ],
  standalone: true,
})
class TestHostComponent {
  dataSource: ListDataSource<CrawlExecutionStatus, unknown>;
  clicked: CrawlExecutionStatus[] = [];
  hasOverflowActions = () => true;
}

describe('CrawlExecutionStatusListComponent', () => {
  let component: CrawlExecutionStatusListComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusListComponent>;
  const getSeed = vi.fn();

  beforeEach(() => {
    getSeed.mockReset();
    getSeed.mockReturnValue(of(new ConfigObject({meta: new Meta({name: 'https://example.com/seed'})})));
    TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusListComponent, TestHostComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: CrawlExecutionService,
          useValue: {getSeed},
        },
        {
          provide: JobExecutionService,
          useValue: {
            getJob: () => of(new ConfigObject({meta: new Meta({name: 'Daily crawl'})})),
          },
        },
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders the table flush with its scroll container', () => {
    const scroll = fixture.nativeElement.querySelector('.scroll') as HTMLElement;

    expect(getComputedStyle(scroll).padding).toBe('0px');
  });

  it('renders execution metrics and errors in the shared column order', async () => {
    const row = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      seedId: 'seed-1',
      jobId: 'job-1',
      state: CrawlExecutionState.FINISHED,
      documentsCrawled: 1234,
      bytesCrawled: 1500,
    });
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(component.displayedColumns).toEqual([
      'seedId',
      'jobId',
      'state',
      'queueSize',
      'documentsCrawled',
      'bytesCrawled',
      'errorCode',
      'startTime',
      'endTime',
      'action',
    ]);
    const headers = [...fixture.nativeElement.querySelectorAll('th')]
      .map((header: HTMLElement) => header.textContent.trim());
    expect(headers).toEqual([
      'Seed', 'Job', 'State', 'Queue', 'Documents', 'Bytes', 'Error', 'Started', 'Ended', '',
    ]);
    expect(fixture.nativeElement.querySelector('.mat-column-desiredState')).toBeNull();
    expect(fixture.nativeElement.querySelector('td.mat-column-documentsCrawled').textContent.trim())
      .toBe('1,234');
    expect(fixture.nativeElement.querySelector('td.mat-column-bytesCrawled').textContent.trim())
      .toBe('1.5 kB');
  });

  it('uses desired state or missing-value text when an end timestamp is absent', async () => {
    const rows = [
      new CrawlExecutionStatus({
        id: 'requested',
        state: CrawlExecutionState.FETCHING,
        desiredState: CrawlExecutionState.ABORTED_TIMEOUT,
      }),
      new CrawlExecutionStatus({
        id: 'active',
        state: CrawlExecutionState.FETCHING,
      }),
      new CrawlExecutionStatus({
        id: 'terminal',
        state: CrawlExecutionState.FAILED,
      }),
      new CrawlExecutionStatus({
        id: 'undefined',
        state: CrawlExecutionState.UNDEFINED,
      }),
    ];
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(...rows),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const endCells = [...fixture.nativeElement.querySelectorAll('td.mat-column-endTime')]
      .map((cell: HTMLElement) => cell.textContent.trim());
    expect(endCells).toEqual(['Aborted after timeout', '', 'Not available', 'Not available']);
  });

  it('renders the seed as plain row content and keeps row navigation', async () => {
    const row = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      seedId: 'seed-1',
      jobId: 'job-1',
      state: CrawlExecutionState.FINISHED,
    });
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    const clicked: CrawlExecutionStatus[] = [];
    component.rowClick.subscribe(item => clicked.push(item));

    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const itemRow = fixture.nativeElement.querySelector('.item-row') as HTMLElement;
    const seedCell = itemRow.querySelector('td') as HTMLElement;
    expect(seedCell.textContent).toContain('https://example.com/seed');
    expect(seedCell.querySelector('a')).toBeNull();

    itemRow.click();
    itemRow.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(clicked).toEqual([row, row]);
  });

  it('renders a deleted seed gracefully with its ID available to users', async () => {
    getSeed.mockReturnValue(of(null));
    const row = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      seedId: 'deleted-seed-id',
      jobId: 'job-1',
      state: CrawlExecutionState.FINISHED,
    });
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);

    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const reference = fixture.nativeElement.querySelector('.deleted-reference') as HTMLElement;
    expect(reference.textContent.trim()).toBe('Deleted seed');
    expect(reference.getAttribute('aria-label')).toBe('Deleted seed. Seed ID: deleted-seed-id');
    expect(component.deletedSeedTooltip(row.seedId)).toBe('Deleted seed ID: deleted-seed-id');
    expect(getComputedStyle(reference).fontStyle).toBe('italic');
  });

  it('keeps log destinations inside the overflow menu', async () => {
    const hostFixture = TestBed.createComponent(TestHostComponent);
    const running = new CrawlExecutionStatus({
      id: 'running',
      state: CrawlExecutionState.FETCHING,
    });
    const finished = new CrawlExecutionStatus({
      id: 'finished',
      state: CrawlExecutionState.FINISHED,
    });
    hostFixture.componentInstance.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(running, finished),
      destroyRef: hostFixture.componentRef.injector.get(DestroyRef),
    });
    hostFixture.detectChanges();
    await hostFixture.whenStable();
    hostFixture.detectChanges();

    const rows = hostFixture.nativeElement.querySelectorAll('.item-row') as NodeListOf<HTMLElement>;
    expect(rows.length).toBe(2);
    expect(rows[0].querySelectorAll('.page-log-action').length).toBe(0);
    expect(rows[0].querySelectorAll('.crawl-log-action').length).toBe(0);
    expect(rows[1].querySelectorAll('.page-log-action').length).toBe(0);
    expect(rows[1].querySelectorAll('.crawl-log-action').length).toBe(0);
    expect(rows[0].querySelector('[aria-label="More actions"]')).not.toBeNull();
    expect(rows[1].querySelector('[aria-label="More actions"]')).not.toBeNull();
    expect(hostFixture.nativeElement.textContent).not.toContain('Go to seed');
    expect(hostFixture.nativeElement.textContent).not.toContain('Go to crawljob');

    (rows[0].querySelector('[aria-label="More actions"]') as HTMLElement).click();
    hostFixture.detectChanges();
    await hostFixture.whenStable();
    const menu = document.querySelector('.mat-mdc-menu-panel') as HTMLElement;
    expect(menu.querySelectorAll('.page-log-action')).toHaveLength(1);
    expect(menu.querySelectorAll('.crawl-log-action')).toHaveLength(1);
    expect(menu.textContent).toContain('Abort running');

    const list = hostFixture.debugElement.query(By.directive(CrawlExecutionStatusListComponent))
      .componentInstance as CrawlExecutionStatusListComponent;
    list.onRowClick(running, {
      target: rows[0].querySelector('[aria-label="More actions"]'),
    } as unknown as Event);
    expect(hostFixture.componentInstance.clicked).toEqual([]);
  });
});
