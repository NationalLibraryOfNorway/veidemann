import {DOCUMENT} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, Input, OnDestroy, ViewChild} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatMenuModule, MatMenuTrigger} from '@angular/material/menu';

@Component({
  selector: 'app-detail-overflow',
  templateUrl: './detail-overflow.component.html',
  styleUrls: ['./detail-overflow.component.scss'],
  imports: [MatButtonModule, MatIcon, MatMenuModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class DetailOverflowComponent implements OnDestroy {
  private readonly document = inject(DOCUMENT);
  private panel: HTMLElement | null = null;
  private readonly handleKeydown = (event: KeyboardEvent): void => {
    if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return;
    const items = this.menuItems();
    if (!items.length) return;
    const current = items.indexOf(this.document.activeElement as HTMLElement);
    const target = event.key === 'Home' ? 0
      : event.key === 'End' ? items.length - 1
        : event.key === 'ArrowUp' ? (current <= 0 ? items.length - 1 : current - 1)
          : (current + 1) % items.length;
    event.preventDefault();
    event.stopImmediatePropagation();
    items[target].focus();
  };

  @Input() label = $localize`:@@detailActionsMenuLabel:More actions`;
  @ViewChild(MatMenuTrigger) private trigger!: MatMenuTrigger;

  onMenuOpened(): void {
    setTimeout(() => {
      this.panel = this.document.getElementById(this.trigger.menu.panelId);
      this.panel?.addEventListener('keydown', this.handleKeydown, true);
      this.menuItems()[0]?.focus();
    });
  }

  onMenuClosed(): void {
    this.panel?.removeEventListener('keydown', this.handleKeydown, true);
    this.panel = null;
  }

  ngOnDestroy(): void {
    this.onMenuClosed();
  }

  private menuItems(): HTMLElement[] {
    return this.panel
      ? [...this.panel.querySelectorAll<HTMLElement>(
        '.mat-mdc-menu-item:not([disabled]):not([aria-disabled="true"])'
      )]
      : [];
  }
}
