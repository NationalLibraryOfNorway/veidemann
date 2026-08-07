import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {of} from 'rxjs';

import {ApiError, ConfigObject, CrawlExecutionState, CrawlExecutionStatus, Meta} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';

@Component({
  template: `
    <app-crawl-execution-status [crawlExecutionStatus]="status">
      <span cardHeaderHelpers class="projected-helper">Helper</span>
      <span detailActions class="projected-action">Action</span>
    </app-crawl-execution-status>
  `,
  imports: [CrawlExecutionStatusComponent],
  standalone: true,
})
class CrawlExecutionStatusHostComponent {
  status = new CrawlExecutionStatus();
}

describe('CrawlExecutionStatusComponent', () => {
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusComponent, CrawlExecutionStatusHostComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: CrawlExecutionService,
          useValue: {
            getSeed: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'Example seed'})})),
          },
        },
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'Example job'})})),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
  });

  function render(status: CrawlExecutionStatus): void {
    fixture.componentRef.setInput('crawlExecutionStatus', status);
    fixture.detectChanges();
  }

  it('uses semantic descriptions and shows identifiers, unavailable times, and all zero metrics', async () => {
    render(new CrawlExecutionStatus({
      id: 'crawl-execution-id-that-can-wrap',
      seedId: 'seed-id-that-can-wrap',
      jobId: 'job-id-that-can-wrap',
      jobExecutionId: 'parent-execution-id-that-can-wrap',
      state: CrawlExecutionState.FINISHED,
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('table')).toBeNull();
    expect(fixture.nativeElement.querySelector('dl.description-list')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('crawl-execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).toContain('parent-execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).toContain('Example seed');
    expect(fixture.nativeElement.textContent).toContain('Example job');
    expect(fixture.nativeElement.querySelectorAll('.metric-tile').length).toBe(7);
    expect(fixture.nativeElement.textContent.match(/Not available/g)?.length).toBe(4);
    expect(fixture.nativeElement.querySelector('.error-card')).toBeNull();
  });

  it.each([
    [CrawlExecutionState.FETCHING, 'state-active'],
    [CrawlExecutionState.CREATED, 'state-waiting'],
    [CrawlExecutionState.SLEEPING, 'state-waiting'],
    [CrawlExecutionState.FINISHED, 'state-finished'],
    [CrawlExecutionState.ABORTED_TIMEOUT, 'state-error'],
    [CrawlExecutionState.ABORTED_SIZE, 'state-error'],
    [CrawlExecutionState.ABORTED_MANUAL, 'state-error'],
    [CrawlExecutionState.FAILED, 'state-error'],
    [CrawlExecutionState.DIED, 'state-error'],
    [CrawlExecutionState.UNDEFINED, 'state-neutral'],
  ])('uses the semantic badge treatment for state %s', (state, expectedClass) => {
    render(new CrawlExecutionStatus({state}));

    const badge = fixture.nativeElement.querySelector('.state-badge') as HTMLElement;
    expect(badge.classList.contains(expectedClass)).toBe(true);
    expect(badge.querySelector('mat-icon')).not.toBeNull();
    expect(badge.textContent.trim().length).toBeGreaterThan(0);
  });

  it('renders message-only errors and allows long details to retain wrapping whitespace', () => {
    render(new CrawlExecutionStatus({error: new ApiError()}));
    expect(fixture.nativeElement.querySelector('.error-card')).toBeNull();

    render(new CrawlExecutionStatus({
      error: new ApiError({detail: 'first line\nsecond line'}),
    }));
    const detail = fixture.nativeElement.querySelector('.error-text') as HTMLElement;
    expect(detail.textContent).toContain('first line\nsecond line');
  });

  it('projects helpers into the overview header and destructive actions below the card', () => {
    const hostFixture = TestBed.createComponent(CrawlExecutionStatusHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('.overview-card mat-card-header') as HTMLElement;
    expect(header.querySelector('.projected-helper')).not.toBeNull();
    expect(header.querySelector('.projected-action')).toBeNull();
    expect(hostFixture.nativeElement.querySelector('.detail-actions .projected-action')).not.toBeNull();
  });
});
