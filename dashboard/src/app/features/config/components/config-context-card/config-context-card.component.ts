import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {LabelDisplayComponent} from '../../../../shared/components';

import {
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  Kind,
  RobotsPolicy,
  RotationPolicy,
} from '../../../../shared/models';

@Component({
  selector: 'app-config-context-card',
  templateUrl: './config-context-card.component.html',
  styleUrls: ['./config-context-card.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LabelDisplayComponent, MatButtonModule, MatCardModule, MatChipsModule, MatIcon, RouterLink],
  standalone: true,
})
export class ConfigContextCardComponent {
  readonly BrowserScriptType = BrowserScriptType;
  readonly Kind = Kind;
  readonly RobotsPolicy = RobotsPolicy;
  readonly RotationPolicy = RotationPolicy;

  @Input({required: true}) configRef: ConfigRef;
  @Input() configObject: ConfigObject | null = null;
  @Input() unavailable = false;

  get title(): string {
    return this.configObject?.meta?.name || this.configRef.id;
  }

  get subtitle(): string {
    return this.configObject?.meta?.description || '';
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
      default: return '';
    }
  }

  routeFor(configRef: ConfigRef): string[] {
    return ['/config', this.detailPath(configRef.kind), configRef.id];
  }
}
