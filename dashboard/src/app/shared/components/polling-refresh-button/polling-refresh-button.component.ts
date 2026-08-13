import {ChangeDetectionStrategy, Component, DestroyRef, EventEmitter, inject, Input, Output, signal} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatTooltipModule} from '@angular/material/tooltip';
import {interval} from 'rxjs';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-polling-refresh-button',
  templateUrl: './polling-refresh-button.component.html',
  styleUrls: ['./polling-refresh-button.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    MatButtonModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
  ],
})
export class PollingRefreshButtonComponent {
  private readonly destroyRef = inject(DestroyRef);
  private cycleStartedAt = Date.now();

  @Input() intervalMs = 15_000;
  @Output() readonly refresh = new EventEmitter<void>();

  readonly progress = signal(0);
  readonly refreshLabel = $localize`:@@pollingRefreshAction:Refresh now`;

  constructor() {
    interval(250).pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => this.tick());
  }

  refreshNow(): void {
    this.completeCycle();
  }

  private tick(): void {
    const elapsed = Date.now() - this.cycleStartedAt;
    if (elapsed >= this.intervalMs) {
      this.completeCycle();
      return;
    }
    this.progress.set(Math.min(100, elapsed / this.intervalMs * 100));
  }

  private completeCycle(): void {
    this.progress.set(100);
    this.cycleStartedAt = Date.now();
    this.refresh.emit();
  }
}
