import {ChangeDetectionStrategy, Component, Input, signal} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {MatTooltip} from '@angular/material/tooltip';
import {LabelDisplayComponent} from '../../../../shared/components';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {timeToDuration} from '../../../../shared/func';
import {
  Annotation,
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  Kind,
  RobotsPolicy,
  RotationPolicy,
} from '../../../../shared/models';
import {configKindIcon} from '../../func/config-kind-icon';

export interface ScriptAnnotationContext {
  jobRef: ConfigRef;
  jobName: string;
  annotations: Annotation[];
  unavailable: boolean;
}

@Component({
  selector: 'app-config-context-card',
  templateUrl: './config-context-card.component.html',
  styleUrls: ['./config-context-card.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FileSizePipe, LabelDisplayComponent, MatButtonModule, MatCardModule, MatChipsModule, MatIcon, MatTooltip, RouterLink],
  standalone: true,
})
export class ConfigContextCardComponent {
  readonly BrowserScriptType = BrowserScriptType;
  readonly Kind = Kind;
  readonly RobotsPolicy = RobotsPolicy;
  readonly RotationPolicy = RotationPolicy;
  readonly configKindIcon = configKindIcon;

  @Input({required: true}) configRef: ConfigRef;
  @Input() configObject: ConfigObject | null = null;
  @Input() unavailable = false;
  @Input() scriptAnnotationContext: ScriptAnnotationContext | null = null;
  @Input() canReadAnnotations = false;

  readonly annotationsExpanded = signal(false);
  get title(): string {
    return this.configObject?.meta?.name || this.configRef.id;
  }

  get inactive(): boolean {
    return !!(this.configObject?.kind === Kind.SEED && this.configObject.seed.disabled
      || this.configObject?.kind === Kind.CRAWLJOB && this.configObject.crawlJob.disabled);
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

  activeLabel(disabled: boolean): string {
    return disabled ? $localize`:@@commonDeactivated:Deactivated` : $localize`:@@commonActive:Active`;
  }

  onLabelDragStart(event: DragEvent, key: string, value: string): void {
    event.dataTransfer?.setData('text/plain', `${key}:${value}`);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'copy';
    }
  }
}
