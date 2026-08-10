import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatChipSelectionChange} from '@angular/material/chips';
import {of} from 'rxjs';

import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {
  ConfigObject,
  JobExecutionState,
  JobExecutionStatus,
  ListDataSource,
  Meta,
} from '../../../shared/models';
import {JobExecutionService, JobExecutionStatusQuery} from '../../report/services';
import {RunningCrawlsComponent} from './running-crawls.component';

describe('RunningCrawlsComponent', () => {
  let fixture: ComponentFixture<RunningCrawlsComponent>;

  const query: JobExecutionStatusQuery = {
    active: 'startTime',
    direction: 'desc',
    jobId: '',
    startTimeFrom: '',
    startTimeTo: '',
    stateList: [JobExecutionState.RUNNING],
    watch: false,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RunningCrawlsComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'Daily crawl'})})),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RunningCrawlsComponent);
  });

  function render(rows: readonly JobExecutionStatus[] = []): void {
    const dataSource = ListDataSource.fromQuery({
      query$: of(query),
      load: () => of(...rows),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    fixture.componentRef.setInput('dataSource', dataSource);
    fixture.componentRef.setInput('selectedStates', [JobExecutionState.RUNNING]);
    fixture.detectChanges();
  }

  it('renders the uncarded latest-jobs table at its natural height with the requested columns', () => {
    render([new JobExecutionStatus({id: 'execution-1'})]);

    expect(fixture.nativeElement.querySelector('h2').textContent).toBe('Latest crawl jobs');
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelector('.container').classList).toContain('embedded');
    const section = fixture.nativeElement.querySelector('.latest-crawl-jobs') as HTMLElement;
    const header = section.querySelector('.latest-crawl-jobs-header') as HTMLElement;
    const list = section.querySelector('app-job-execution-status-list') as HTMLElement;
    expect(getComputedStyle(section).width).toBe('100%');
    expect(getComputedStyle(header).paddingInline).toBe('16px');
    expect(getComputedStyle(list).width).toBe('100%');
    expect(getComputedStyle(list).borderStyle).toBe('none');
    const headers = [...fixture.nativeElement.querySelectorAll('th')]
      .map((header: HTMLElement) => header.textContent.trim());
    expect(headers).toEqual([
      'Job',
      'State',
      'Documents crawled',
      'Bytes crawled',
      'Start',
      'End',
      'Duration',
    ]);
  });

  it('hides the table when no crawl jobs match', () => {
    render();

    expect(fixture.nativeElement.querySelector('app-job-execution-status-list')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('No crawl jobs match the selected states');
  });

  it('shows job data and emits the whole row for mouse and keyboard navigation', async () => {
    const row = new JobExecutionStatus({
      id: 'execution-1',
      jobId: 'job-1',
      state: JobExecutionState.FINISHED,
      startTime: '2026-08-04T08:00:00.000Z',
      endTime: '2026-08-04T09:02:00.000Z',
      documentsCrawled: 1234,
      bytesCrawled: 1500,
      desiredState: JobExecutionState.ABORTED_MANUAL,
    });
    render([row]);
    await fixture.whenStable();
    fixture.detectChanges();

    const emitted: JobExecutionStatus[] = [];
    fixture.componentInstance.rowClick.subscribe(value => emitted.push(value));
    const itemRow = fixture.nativeElement.querySelector('.item-row') as HTMLElement;
    expect(itemRow.textContent).toContain('Daily crawl');
    expect(itemRow.textContent).toContain('FINISHED');
    const desiredStateBadge = itemRow.querySelector('.desired-state-badge') as HTMLElement;
    expect(desiredStateBadge.textContent.trim()).toBe('ABORTED_MANUAL');
    expect(desiredStateBadge.getAttribute('aria-label')).toBe('Desired state: ABORTED_MANUAL');
    expect(itemRow.querySelector('.mat-column-queueSize')).toBeNull();
    expect(itemRow.textContent).toContain('1,234');
    expect(itemRow.querySelector('.mat-column-bytesCrawled')?.textContent.trim()).toBe('1.5 kB');
    expect(itemRow.textContent).toContain('1hours:2min');

    itemRow.click();
    itemRow.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(emitted).toEqual([row, row]);
  });

  it('starts with only RUNNING selected and maps ABORTED to the aborted state family', () => {
    render();

    const chips = [...fixture.nativeElement.querySelectorAll('mat-chip-option')] as HTMLElement[];
    expect(chips.map(chip => chip.textContent.trim())).toEqual(['RUNNING', 'FINISHED', 'ABORTED', 'FAILED']);
    expect(fixture.componentInstance.stateChips.map(
      chip => fixture.componentInstance.isChipSelected(chip)
    )).toEqual([true, false, false, false]);

    const emitted: (readonly JobExecutionState[])[] = [];
    fixture.componentInstance.selectedStatesChange.subscribe(states => emitted.push(states));
    fixture.componentInstance.onChipSelectionChange(
      fixture.componentInstance.stateChips[2],
      {isUserInput: true, selected: true} as MatChipSelectionChange
    );
    expect(emitted).toEqual([[JobExecutionState.RUNNING, JobExecutionState.ABORTED_MANUAL]]);
  });

  it('allows the final selected state chip to be removed to show all states', () => {
    render();
    const emitted: (readonly JobExecutionState[])[] = [];
    fixture.componentInstance.selectedStatesChange.subscribe(states => emitted.push(states));

    fixture.componentInstance.onChipSelectionChange(
      fixture.componentInstance.stateChips[0],
      {isUserInput: true, selected: false} as MatChipSelectionChange
    );

    expect(emitted).toEqual([[]]);
  });
});
