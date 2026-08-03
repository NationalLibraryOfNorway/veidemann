import { Directive, inject } from '@angular/core';
import { AuthService } from '../../../core';

@Directive()
export abstract class NavigationListComponent {
  protected authService = inject(AuthService);

  get canAdministrate(): boolean {
    return this.authService.isAdmin();
  }

  get canConfigure(): boolean {
    return this.authService.isAdmin() || this.authService.isCurator();
  }

  get canConsult(): boolean {
    return this.authService.isConsultant();
  }
}
