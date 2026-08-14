import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output, inject} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatMenuModule} from '@angular/material/menu';
import {Sort, SortDirection} from '@angular/material/sort';
import {MatTooltip} from '@angular/material/tooltip';
import {Params, RouterLink} from '@angular/router';

import {AuthService} from '../../../../../core/auth';
import {ActionDirective, FilterDirective} from '../../../../../shared/directives';
import {ConfigQuery} from '../../../../../shared/func';
import {ConfigObject, Kind, ListDataSource} from '../../../../../shared/models';
import {ConfigListComponent} from '../../config-list/config-list.component';
import {configKindIcon} from '../../../func/config-kind-icon';

@Component({
  selector: 'app-entity-seed-context',
  templateUrl: './entity-seed-context.component.html',
  styleUrls: ['./entity-seed-context.component.scss'],
  imports: [
    ActionDirective,
    ConfigListComponent,
    FilterDirective,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatTooltip,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class EntitySeedContextComponent {
  protected authService = inject(AuthService);
  readonly Kind = Kind;
  readonly configKindIcon = configKindIcon;

  @Input({required: true}) entity: ConfigObject;
  @Input({required: true}) seedDataSource: ListDataSource<ConfigObject, ConfigQuery>;
  @Input() seedStatus: boolean | null = null;
  @Input() seedSortActive = '';
  @Input() seedSortDirection: SortDirection = '';

  @Output() seedStatusChange = new EventEmitter<boolean | null>();
  @Output() seedSortChange = new EventEmitter<Sort>();
  @Output() createSeed = new EventEmitter<ConfigObject>();
  @Output() openSeed = new EventEmitter<ConfigObject>();
  @Output() editSeed = new EventEmitter<ConfigObject>();
  @Output() runSeed = new EventEmitter<ConfigObject>();
  @Output() cloneSeed = new EventEmitter<ConfigObject>();
  @Output() deleteSeed = new EventEmitter<ConfigObject>();

  get canEditSeed(): boolean {
    return this.authService.canUpdate(Kind.SEED);
  }

  get canCreateSeed(): boolean {
    return this.authService.canCreate(Kind.SEED);
  }

  get canCloneSeed(): boolean {
    return this.authService.canCreate(Kind.SEED);
  }

  get canDeleteSeed(): boolean {
    return this.authService.canDelete(Kind.SEED);
  }

  get canReadCrawlExecutions(): boolean {
    return this.authService.canRead('crawlexecution');
  }

  get canReadSeeds(): boolean {
    return this.authService.canRead(Kind.SEED);
  }

  get canRunSeed(): boolean {
    return this.authService.canRunCrawl(Kind.SEED);
  }

  getJobRefListQueryParams(seed: ConfigObject): Params {
    return {crawl_job_id: seed.seed.jobRefList.map(jobRef => jobRef.id)};
  }
}
