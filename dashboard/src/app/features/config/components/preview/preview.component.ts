import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {ActivatedRoute, Router} from '@angular/router';
import {ErrorService} from '../../../../core';
import {SeedMetaPreviewComponent} from '../seed-meta/seed-meta-preview/seed-meta-preview.component';
import {MetaPreviewComponent} from '../meta/meta-preview/meta-preview.component';
import {SeedPreviewComponent} from '../seed/seed-preview/seed-preview.component';
import {CollectionPreviewComponent} from '../collection/collection-preview/collection-preview.component';
import {CrawljobPreviewComponent} from '../crawljobs/crawljob-preview/crawljob-preview.component';
import {SchedulePreviewComponent} from '../schedule/schedule-preview/schedule-preview.component';
import {CrawlconfigPreviewComponent} from '../crawlconfig/crawlconfig-preview/crawlconfig-preview.component';
import {
  CrawlhostgroupconfigPreviewComponent
} from '../crawlhostgroupconfig/crawlhostgroupconfig-preview/crawlhostgroupconfig-preview.component';
import {BrowserconfigPreviewComponent} from '../browserconfig/browserconfig-preview/browserconfig-preview.component';
import {BrowserscriptPreviewComponent} from '../browserscript/browserscript-preview/browserscript-preview.component';
import {
  PolitenessconfigPreviewComponent
} from '../politenessconfig/politenessconfig-preview/politenessconfig-preview.component';
import {ShortcutListComponent} from '../shortcut/shortcut-list/shortcut-list.component';

@Component({
  selector: 'app-preview',
  templateUrl: './preview.component.html',
  styleUrls: ['./preview.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    SeedMetaPreviewComponent,
    MetaPreviewComponent,
    SeedPreviewComponent,
    CollectionPreviewComponent,
    CrawljobPreviewComponent,
    SchedulePreviewComponent,
    CrawlconfigPreviewComponent,
    CrawlhostgroupconfigPreviewComponent,
    BrowserconfigPreviewComponent,
    BrowserscriptPreviewComponent,
    PolitenessconfigPreviewComponent,
    ShortcutListComponent
  ],
  standalone: true
})
export class PreviewComponent {
  readonly Kind = Kind;

  @Input()
  configObject: ConfigObject;

  constructor(private router: Router,
              private route: ActivatedRoute,
              private errorService: ErrorService) {
  }

  onEditConfig(configObject: ConfigObject) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {id: configObject.id},
    }).catch(error => this.errorService.dispatch(error));
  }
}
