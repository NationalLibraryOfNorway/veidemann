import { Component, OnInit, ChangeDetectionStrategy, inject } from '@angular/core';
import {PageLogService} from '../../services';
import {DetailDirective} from '../../directives';
import {PageLog} from '../../../../shared/models';
import {Observable} from 'rxjs';
import {map, mergeMap} from 'rxjs/operators';
import {PageLogStatusComponent} from '../../components';
import {PageLogShortcutsComponent} from '../../components/page-log-shortcuts/page-log-shortcuts.component';
import {CommonModule} from '@angular/common';

@Component({
    selector: 'app-crawl-log-detail',
    templateUrl: './page-log-detail.component.html',
    styleUrls: ['../detail-layout.scss'],
  imports: [
    PageLogStatusComponent,
    PageLogShortcutsComponent,
    CommonModule,
  ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true
})
export class PageLogDetailComponent extends DetailDirective<PageLog> implements OnInit {
  protected override service = inject(PageLogService);

  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<PageLog> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      mergeMap(query => this.service.get(query)),
    );

    this.item$ = item$;
  }
}
