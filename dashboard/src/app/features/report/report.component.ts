import {ChangeDetectionStrategy, Component} from '@angular/core';
import {MatSidenavModule} from '@angular/material/sidenav';
import {RouterModule} from '@angular/router';
import {ReportNavigationListComponent} from './containers/';

@Component({
    selector: 'app-report',
    templateUrl: './report.component.html',
    styleUrls: ['./report.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
    imports: [
      RouterModule,
      MatSidenavModule,
      ReportNavigationListComponent,
    ],

})
export class ReportComponent {
}
