import {Component} from '@angular/core';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {Observable} from 'rxjs';
import {AbilityService} from '@casl/angular';

@Component({
    selector: 'app-report-navigation-list',
    templateUrl: './report-navigation-list.component.html',
    styleUrls: ['../../../commons/components/navigation-list/navigation-list.component.scss'],
    standalone: true
})
export class ReportNavigationListComponent extends NavigationListComponent {
  readonly ability$: Observable<any>;

  constructor(protected override authService: AuthService,
              private abilityService: AbilityService<any>) {
    super(authService);
    this.ability$ = this.abilityService.ability$;
  }
}
