import {ChangeDetectionStrategy, Component, computed, inject, Signal} from '@angular/core';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter, map, startWith} from 'rxjs/operators';
import {toSignal} from '@angular/core/rxjs-interop';
import {SectionHeaderComponent} from '../../shared/components/section-header/section-header.component';

@Component({
    selector: 'app-report',
    templateUrl: './report.component.html',
    styleUrls: ['./report.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
    imports: [
      RouterOutlet,
      SectionHeaderComponent,
    ],

})
export class ReportComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly reportsLabel = $localize`:@@reportPageTitle:Reports`;
  readonly sectionPath: Signal<string>;
  readonly sectionTitle: Signal<string>;
  readonly isDetail: Signal<boolean>;
  readonly listLink: Signal<string[]>;
  readonly backLink: Signal<string[]>;

  constructor() {
    const routeState = toSignal(
      this.router.events.pipe(
        filter(event => event instanceof NavigationEnd),
        startWith(null),
        map(() => ({
          path: this.route.firstChild?.snapshot.url[0]?.path ?? '',
          detail: !!this.route.firstChild?.snapshot.paramMap.get('id'),
        })),
      ),
      {requireSync: true}
    );

    this.sectionPath = computed(() => routeState().path);
    this.sectionTitle = computed(() => this.getSectionTitle(this.sectionPath()));
    this.isDetail = computed(() => routeState().detail);
    this.listLink = computed(() => ['/report', this.sectionPath()]);
    this.backLink = computed(() => this.isDetail() ? this.listLink() : ['/report']);
  }

  private getSectionTitle(path: string): string {
    switch (path) {
      case 'jobexecution':
        return $localize`:@@reportNavigationLinkJobExecution:JobExecution`;
      case 'crawlexecution':
        return $localize`:@@reportNavigationLinkCrawlExecution:CrawlExecution`;
      case 'pagelog':
        return $localize`:@@reportNavigationLinkPageLog:PageLog`;
      case 'crawllog':
        return $localize`:@@reportNavigationLinkCrawlLog:CrawlLog`;
      default:
        return this.reportsLabel;
    }
  }
}
