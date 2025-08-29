import {Component, OnInit} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {PageLogService} from '../../services';
import {DetailDirective} from '../../directives';
import {PageLog} from '../../../../shared/models';
import {Observable} from 'rxjs';
import {map, mergeMap} from 'rxjs/operators';
import { PageLogStatusComponent } from '../../components';
import { PageLogShortcutsComponent } from '../../components/page-log-shortcuts/page-log-shortcuts.component';
import { CommonModule } from '@angular/common';
import {FlexLayoutModule} from '@angular/flex-layout';

@Component({
    selector: 'app-crawl-log-detail',
    templateUrl: './page-log-detail.component.html',
    styleUrls: ['./page-log-detail.component.scss'],
  imports: [
    PageLogStatusComponent,
    PageLogShortcutsComponent,
    CommonModule,
    FlexLayoutModule,
  ],
    standalone: true
})
export class PageLogDetailComponent extends DetailDirective<PageLog> implements OnInit {

  constructor(protected override route: ActivatedRoute,
              protected override service: PageLogService) {
    super(route, service);
  }

  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<PageLog> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      mergeMap(query => this.service.get(query)),
    );

    this.item$ = item$;
  }
}
