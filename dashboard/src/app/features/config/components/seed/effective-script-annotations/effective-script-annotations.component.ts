import {ChangeDetectionStrategy, Component, Input, OnChanges, SimpleChanges, inject, signal} from '@angular/core';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {take} from 'rxjs/operators';

import {LabelDisplayComponent} from '../../../../../shared/components';
import {Annotation, ConfigObject, ConfigRef, Kind} from '../../../../../shared/models';
import {ConfigService} from '../../../../../shared/services';
import {SCRIPT_ANNOTATION_DRAG_TYPE, ScriptAnnotationContext} from '../../script-annotation-context';

interface SelectedCrawlJob {
  id: string;
  name: string;
}

@Component({
  selector: 'app-effective-script-annotations',
  templateUrl: './effective-script-annotations.component.html',
  styleUrls: ['./effective-script-annotations.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LabelDisplayComponent, MatChipsModule, MatIcon, MatProgressSpinnerModule],
  standalone: true,
})
export class EffectiveScriptAnnotationsComponent implements OnChanges {
  private readonly configService = inject(ConfigService);
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  @Input() jobIds: string[] = [];
  @Input() crawlJobs: ConfigObject[] = [];
  @Input() seedId = '';

  readonly canReadAnnotations = this.abilityService.can('read', 'annotation');
  readonly expandedJobIds = signal<Set<string>>(new Set());
  readonly loadingJobIds = signal<Set<string>>(new Set());
  readonly contexts = signal<Map<string, ScriptAnnotationContext>>(new Map());

  get selectedCrawlJobs(): SelectedCrawlJob[] {
    const seen = new Set<string>();
    return (this.jobIds ?? []).filter(id => {
      if (!id || seen.has(id)) {
        return false;
      }
      seen.add(id);
      return true;
    }).map(id => ({
      id,
      name: this.crawlJobs?.find(job => job.id === id)?.meta?.name?.trim() || id,
    }));
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['seedId']) {
      this.contexts.set(new Map());
      this.loadingJobIds.set(new Set());
    }

    if (changes['jobIds']) {
      const selected = new Set(this.jobIds ?? []);
      this.expandedJobIds.update(expanded => new Set([...expanded].filter(id => selected.has(id))));
    }
  }

  isExpanded(jobId: string): boolean {
    return this.expandedJobIds().has(jobId);
  }

  isLoading(jobId: string): boolean {
    return this.loadingJobIds().has(jobId);
  }

  contextFor(jobId: string): ScriptAnnotationContext | undefined {
    return this.contexts().get(jobId);
  }

  toggle(job: SelectedCrawlJob): void {
    if (this.isExpanded(job.id)) {
      this.expandedJobIds.update(expanded => this.without(expanded, job.id));
      return;
    }

    this.expandedJobIds.update(expanded => new Set(expanded).add(job.id));
    if (!this.contextFor(job.id) && !this.isLoading(job.id)) {
      this.load(job);
    }
  }

  onChipKeydown(event: KeyboardEvent, job: SelectedCrawlJob): void {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    this.toggle(job);
  }

  onAnnotationDragStart(event: DragEvent, annotation: Annotation): void {
    if (!event.dataTransfer) {
      return;
    }
    event.dataTransfer.effectAllowed = 'copy';
    event.dataTransfer.setData(SCRIPT_ANNOTATION_DRAG_TYPE, JSON.stringify({
      key: annotation.key,
      value: annotation.value,
    }));
    event.dataTransfer.setData('text/plain', `${annotation.key}:${annotation.value}`);
  }

  annotationDragLabel(annotation: Annotation): string {
    return $localize`:@@effectiveScriptAnnotationDragLabel:Drag ${annotation.key}\:${annotation.value} to script annotations`;
  }

  private load(job: SelectedCrawlJob): void {
    this.loadingJobIds.update(loading => new Set(loading).add(job.id));
    this.configService.getScriptAnnotations(job.id, this.seedId || undefined).pipe(take(1)).subscribe({
      next: annotations => this.setContext(job, annotations, false),
      error: () => this.setContext(job, [], true),
    });
  }

  private setContext(job: SelectedCrawlJob, annotations: Annotation[], unavailable: boolean): void {
    this.contexts.update(contexts => {
      const next = new Map(contexts);
      next.set(job.id, {
        jobRef: new ConfigRef({kind: Kind.CRAWLJOB, id: job.id}),
        jobName: job.name,
        annotations,
        unavailable,
      });
      return next;
    });
    this.loadingJobIds.update(loading => this.without(loading, job.id));
  }

  private without(values: Set<string>, value: string): Set<string> {
    const next = new Set(values);
    next.delete(value);
    return next;
  }
}
