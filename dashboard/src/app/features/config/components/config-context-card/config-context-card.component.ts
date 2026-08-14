import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {timeToDuration} from '../../../../shared/func';
import {
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  Kind,
  RobotsPolicy,
  RotationPolicy,
} from '../../../../shared/models';
import {configKindIcon} from '../../func/config-kind-icon';
import {ConfigRelationRole, ConfigRelationSource} from '../../func';

@Component({
  selector: 'app-config-context-card',
  templateUrl: './config-context-card.component.html',
  styleUrls: ['./config-context-card.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FileSizePipe, MatIcon, RouterLink],
  standalone: true,
})
export class ConfigContextCardComponent {
  readonly BrowserScriptType = BrowserScriptType;
  readonly Kind = Kind;
  readonly RobotsPolicy = RobotsPolicy;
  readonly RotationPolicy = RotationPolicy;
  readonly configKindIcon = configKindIcon;

  @Input({required: true}) configRef: ConfigRef;
  @Input() relationRole: ConfigRelationRole = 'browser-script';
  @Input() relationSource: ConfigRelationSource = 'direct';
  @Input() configObject: ConfigObject | null = null;
  @Input() unavailable = false;
  get title(): string {
    return this.configObject?.meta?.name || this.configRef.id;
  }

  get inactive(): boolean {
    return !!(this.configObject?.kind === Kind.SEED && this.configObject.seed.disabled
      || this.configObject?.kind === Kind.CRAWLJOB && this.configObject.crawlJob.disabled);
  }

  kindLabel(kind: Kind): string {
    switch (kind) {
      case Kind.CRAWLENTITY: return $localize`:@@configKindEntity:Entity`;
      case Kind.SEED: return $localize`:@@configKindSeed:Seed`;
      case Kind.CRAWLJOB: return $localize`:@@configKindCrawlJob:Crawl job`;
      case Kind.CRAWLSCHEDULECONFIG: return $localize`:@@configKindSchedule:Schedule`;
      case Kind.CRAWLCONFIG: return $localize`:@@configKindCrawlConfig:Crawl config`;
      case Kind.COLLECTION: return $localize`:@@configKindCollection:Collection`;
      case Kind.BROWSERCONFIG: return $localize`:@@configKindBrowserConfig:Browser config`;
      case Kind.BROWSERSCRIPT: return $localize`:@@configKindBrowserScript:Browser script`;
      case Kind.POLITENESSCONFIG: return $localize`:@@configKindPoliteness:Politeness config`;
      default: return $localize`:@@configKindConfiguration:Configuration`;
    }
  }

  detailPath(kind: Kind): string {
    switch (kind) {
      case Kind.CRAWLENTITY: return 'entity';
      case Kind.SEED: return 'seed';
      case Kind.CRAWLJOB: return 'crawljobs';
      case Kind.CRAWLCONFIG: return 'crawlconfig';
      case Kind.CRAWLSCHEDULECONFIG: return 'schedule';
      case Kind.BROWSERCONFIG: return 'browserconfig';
      case Kind.POLITENESSCONFIG: return 'politenessconfig';
      case Kind.BROWSERSCRIPT: return 'browserscript';
      case Kind.COLLECTION: return 'collection';
      case Kind.CRAWLHOSTGROUPCONFIG: return 'crawlhostgroupconfig';
      case Kind.ROLEMAPPING: return 'rolemapping';
      default: return '';
    }
  }

  duration(value: number, unit: 'ms' | 's'): string {
    return timeToDuration(value, unit).replaceAll(':', ' ')
      .replace('days', ' d').replace('hours', ' h');
  }

  enabledLabel(value: boolean): string {
    return value ? $localize`:@@commonEnabled:Enabled` : $localize`:@@commonDisabled:Disabled`;
  }

  yesNoLabel(value: boolean): string {
    return value ? $localize`:@@commonYes:Yes` : $localize`:@@commonNo:No`;
  }

}
