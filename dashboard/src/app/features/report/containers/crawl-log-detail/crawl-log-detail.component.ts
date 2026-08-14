import {CommonModule} from '@angular/common';
import { Component, OnInit, ChangeDetectionStrategy, inject } from '@angular/core';
import {Observable} from 'rxjs';
import {map, mergeMap} from 'rxjs/operators';
import {CrawlLog} from '../../../../shared/models';
import {CrawlLogStatusComponent} from '../../components';
import {DetailDirective} from '../../directives';
import {CrawlLogService} from '../../services';

@Component({
  selector: 'app-crawl-log-detail',
  templateUrl: './crawl-log-detail.component.html',
  styleUrls: ['../detail-layout.scss'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    CrawlLogStatusComponent,
  ]
})
export class CrawlLogDetailComponent extends DetailDirective<CrawlLog> implements OnInit {
  protected override service = inject(CrawlLogService);

  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<CrawlLog> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      mergeMap(query => this.service.get(query)),
    );

    this.item$ = item$;
  }
}
