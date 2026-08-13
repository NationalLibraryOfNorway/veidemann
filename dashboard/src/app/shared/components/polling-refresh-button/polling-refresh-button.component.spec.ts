import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {PollingRefreshButtonComponent} from './polling-refresh-button.component';

describe('PollingRefreshButtonComponent', () => {
  let fixture: ComponentFixture<PollingRefreshButtonComponent>;

  beforeEach(async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-13T12:00:00Z'));
    await TestBed.configureTestingModule({
      imports: [PollingRefreshButtonComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();
    fixture = TestBed.createComponent(PollingRefreshButtonComponent);
    fixture.detectChanges();
  });

  afterEach(() => vi.useRealTimers());

  it('fills the determinate ring and refreshes automatically every 15 seconds', async () => {
    const refreshed = vi.fn();
    fixture.componentInstance.refresh.subscribe(refreshed);

    await vi.advanceTimersByTimeAsync(7_500);
    expect(fixture.componentInstance.progress()).toBeCloseTo(50, 0);
    expect(refreshed).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(7_500);
    expect(fixture.componentInstance.progress()).toBe(100);
    expect(refreshed).toHaveBeenCalledOnce();
    const spinner = fixture.nativeElement.querySelector('mat-progress-spinner') as HTMLElement;
    expect(spinner.getAttribute('mode')).toBe('determinate');
  });

  it('refreshes immediately on click and restarts the countdown', async () => {
    const refreshed = vi.fn();
    fixture.componentInstance.refresh.subscribe(refreshed);
    await vi.advanceTimersByTimeAsync(5_000);

    (fixture.nativeElement.querySelector('button') as HTMLButtonElement).click();

    expect(refreshed).toHaveBeenCalledOnce();
    expect(fixture.componentInstance.progress()).toBe(100);
    await vi.advanceTimersByTimeAsync(250);
    expect(fixture.componentInstance.progress()).toBeLessThan(5);
  });
});
