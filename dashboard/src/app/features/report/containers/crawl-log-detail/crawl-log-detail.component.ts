import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs';
import { map, mergeMap } from 'rxjs/operators';
import { CrawlLog } from '../../../../shared/models';
import { CrawlLogStatusComponent } from '../../components';
import { CrawlLogShortcutsComponent } from '../../components/crawl-log-shortcuts/crawl-log-shortcuts.component';
import { DetailDirective } from '../../directives';
import { CrawlLogService } from '../../services';

@Component({
    selector: 'app-crawl-log-detail',
    templateUrl: './crawl-log-detail.component.html',
    styleUrls: ['./crawl-log-detail.component.css'],
    standalone: true,
    imports: [
      CommonModule,
      MatIconModule,
      CrawlLogStatusComponent,
      CrawlLogShortcutsComponent,
    ]
})
export class CrawlLogDetailComponent extends DetailDirective<CrawlLog> implements OnInit {

  constructor(protected override route: ActivatedRoute,
              protected crawlLogService: CrawlLogService) {
    super(route, crawlLogService);
  }

  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<CrawlLog> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      mergeMap(query => this.service.get(query)),
    );

    this.item$ = item$;
  }
}
