import {Component} from '@angular/core';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {Observable} from 'rxjs';
import {AbilityService} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {MatListItem, MatNavList} from '@angular/material/list';
import {RouterLink, RouterLinkActive} from '@angular/router';

@Component({
  selector: 'app-report-navigation-list',
  templateUrl: './report-navigation-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-list/navigation-list.component.scss'],
  imports: [
    AsyncPipe,
    MatIcon,
    MatListItem,
    MatNavList,
    RouterLink,
    RouterLinkActive
  ],
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
