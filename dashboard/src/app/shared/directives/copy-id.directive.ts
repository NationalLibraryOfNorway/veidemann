import {Clipboard} from '@angular/cdk/clipboard';
import {Directive, HostListener, Input, inject} from '@angular/core';

import {SnackBarService} from '../../core';

@Directive({
  selector: '[appCopyId]',
  standalone: true,
})
export class CopyIdDirective {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBarService = inject(SnackBarService);

  @Input({required: true}) appCopyId = '';

  @HostListener('click')
  copy(): void {
    if (!this.appCopyId) {
      return;
    }

    if (this.clipboard.copy(this.appCopyId)) {
      this.snackBarService.openSnackBar($localize`:@@copyIdSuccess:ID copied`);
      return;
    }

    this.snackBarService.openError(new Error($localize`:@@copyIdFailure:Could not copy ID`));
  }

  @HostListener('keydown', ['$event'])
  onKeydown(event: KeyboardEvent): void {
    if (event.currentTarget instanceof HTMLButtonElement
      || event.currentTarget instanceof HTMLAnchorElement
      || (event.key !== 'Enter' && event.key !== ' ')) {
      return;
    }

    event.preventDefault();
    this.copy();
  }
}
