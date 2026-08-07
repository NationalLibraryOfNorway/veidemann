import {ComponentFixture, TestBed} from '@angular/core/testing';
import {AbilityServiceSignal} from '@casl/angular';

import {CrawlerStatus, RunStatus} from '../../../shared/models/controller';
import {CrawlerStatusComponent} from './crawlerstatus.component';

describe('CrawlerStatusComponent', () => {
  let fixture: ComponentFixture<CrawlerStatusComponent>;
  let canUpdate: boolean;

  beforeEach(async () => {
    canUpdate = true;

    await TestBed.configureTestingModule({
      imports: [CrawlerStatusComponent],
      providers: [
        {
          provide: AbilityServiceSignal,
          useValue: {can: () => canUpdate},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlerStatusComponent);
  });

  it.each([
    [RunStatus.RUNNING, 'status-running', 'Running', 'Pause crawler'],
    [RunStatus.PAUSE_REQUESTED, 'status-pausing', 'Pausing', null],
    [RunStatus.PAUSED, 'status-paused', 'Paused', 'Resume crawler'],
  ])('renders run state %s with its semantic treatment', (runStatus, cssClass, label, action) => {
    fixture.componentRef.setInput('crawlerStatus', new CrawlerStatus({runStatus}));
    fixture.detectChanges();

    const badge = fixture.nativeElement.querySelector('.status-badge') as HTMLElement;
    const actionButton = fixture.nativeElement.querySelector('.status-action') as HTMLButtonElement | null;

    expect(badge.classList).toContain(cssClass);
    expect(badge.textContent).toContain(label);
    expect(actionButton === null).toBe(action === null);
    if (actionButton) {
      expect(actionButton.tagName).toBe('BUTTON');
      expect(actionButton.textContent).toContain(action);
    }
    expect(fixture.nativeElement.querySelector('.status-menu-trigger')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-menu')).toBeNull();
  });

  it('shows formatted metrics to a read-only user without showing an action', () => {
    canUpdate = false;
    fixture.componentRef.setInput('crawlerStatus', new CrawlerStatus({
      runStatus: RunStatus.RUNNING,
      queueSize: 1234,
      busyCrawlHostGroupCount: 8,
    }));
    fixture.detectChanges();

    const values = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.metric-value'))
      .map(element => element.textContent.trim());

    expect(values).toEqual(['1,234', '8']);
    expect(fixture.nativeElement.querySelector('.status-overview').getAttribute('aria-label'))
      .toBe('Crawler overview');
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelectorAll('.metric-badge').length).toBe(2);
    expect(fixture.nativeElement.querySelector('.status-action')).toBeNull();
  });

  it.each([
    [RunStatus.RUNNING, true],
    [RunStatus.PAUSED, false],
  ])('emits the existing status-change value for run state %s', (runStatus, expected) => {
    const statusChange = vi.fn();
    fixture.componentInstance.changeRunStatus.subscribe(statusChange);
    fixture.componentRef.setInput('crawlerStatus', new CrawlerStatus({runStatus}));
    fixture.detectChanges();

    const actionButton = fixture.nativeElement.querySelector('.status-action') as HTMLButtonElement;
    actionButton.click();

    expect(statusChange).toHaveBeenCalledWith(expected);
  });
});
