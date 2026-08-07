import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {JobExecutionStatus, Kind} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {JobExecutionShortcutHelpersComponent} from './job-execution-shortcuts.component';

describe('JobExecutionShortcutHelpersComponent', () => {
  let can: ReturnType<typeof vi.fn>;
  let fixture: ComponentFixture<JobExecutionShortcutHelpersComponent>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    await TestBed.configureTestingModule({
      imports: [JobExecutionShortcutHelpersComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(JobExecutionShortcutHelpersComponent);
  });

  function render(): JobExecutionStatus {
    const status = new JobExecutionStatus({id: 'execution-1', jobId: 'job-1'});
    fixture.componentRef.setInput('jobExecutionStatus', status);
    fixture.detectChanges();
    return status;
  }

  it('shows links in one accessible shortcut set without a visible category label', () => {
    render();

    expect(fixture.nativeElement.textContent).not.toContain('more_vert');
    expect(fixture.nativeElement.textContent).not.toContain('Related resources');
    expect(fixture.nativeElement.querySelector('button[aria-label="More actions"]')).toBeNull();
    const links = fixture.nativeElement.querySelectorAll('a') as NodeListOf<HTMLAnchorElement>;
    const chipSet = fixture.nativeElement.querySelector('mat-chip-set') as HTMLElement;
    expect(links.length).toBe(2);
    expect(fixture.nativeElement.querySelectorAll('mat-chip-set').length).toBe(1);
    expect(chipSet.getAttribute('aria-label')).toBe('Report shortcuts');
    expect([...links].every(link => link.hasAttribute('mat-chip'))).toBe(true);
    expect(links[0].getAttribute('href')).toBe('/config/crawljobs/job-1');
    expect(links[1].getAttribute('href')).toContain('/report/crawlexecution');
    expect(links[1].getAttribute('href')).toContain('job_id=job-1');
    expect(links[1].getAttribute('href')).toContain('job_execution_id=execution-1');
  });

  it('collapses all helper groups when no link is permitted', () => {
    can.mockReturnValue(false);
    render();

    expect(fixture.nativeElement.querySelector('mat-chip-set')).toBeNull();
  });

  it('lets list contexts link to the current Job Execution without the plural Crawl Executions link', () => {
    fixture.componentRef.setInput('showJobExecution', true);
    fixture.componentRef.setInput('showCrawlExecutions', false);
    render();

    const hrefs = [...fixture.nativeElement.querySelectorAll('a')]
      .map((link: HTMLAnchorElement) => link.getAttribute('href'));
    expect(hrefs).toEqual([
      '/report/jobexecution/execution-1',
      '/config/crawljobs/job-1',
    ]);
  });

  it.each([
    [Kind[Kind.CRAWLJOB], 'Crawl job'],
    ['crawlexecution', 'Crawl executions'],
  ])('applies the read permission for %s to its related link', (deniedSubject, hiddenLabel) => {
    can.mockImplementation((action: string, subject: string) =>
      action !== 'read' || subject !== deniedSubject);
    render();

    expect(fixture.nativeElement.textContent).not.toContain(hiddenLabel);
  });
});
