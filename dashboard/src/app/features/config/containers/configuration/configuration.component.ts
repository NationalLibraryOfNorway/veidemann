import {ChangeDetectionStrategy, Component, computed, DestroyRef, ErrorHandler, OnDestroy, Signal, signal, inject} from '@angular/core';
import {ActivatedRoute, NavigationStart, Router, RouterLink} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';

import {combineLatest, EMPTY, forkJoin, merge, Observable, of, Subject} from 'rxjs';
import {catchError, filter, map, shareReplay, switchMap, take, takeUntil, tap} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {
  Annotation,
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  Kind,
  ListDataSource,
  RobotsPolicy,
  Role,
  RotationPolicy,
  Seed,
  SubCollectionType
} from '../../../../shared/models';
import {AuthService, ControllerApiService, SnackBarService} from '../../../../core';
import {
  BrowserConfigDetailsComponent,
  BrowserScriptDetailsComponent,
  CollectionDetailsComponent,
  ConfigLabelLinksComponent,
  ConfigContextCardComponent,
  CrawlConfigDetailsComponent,
  CrawlExecutionStatusComponent,
  CrawlHostGroupConfigDetailsComponent,
  CrawlJobDetailsComponent,
  DeleteDialogComponent,
  EntityDetailsComponent,
  EntitySeedContextComponent,
  JobStatusComponent,
  Parcel,
  PolitenessConfigDetailsComponent,
  RoleMappingDetailsComponent,
  ScheduleDetailsComponent,
  ScriptAnnotationContext,
  SeedDetailsComponent
} from '../../components';
import {OptionsResolver, OptionsService} from '../../services';
import {RunCrawlDialogComponent} from '../../components/run-crawl-dialog/run-crawl-dialog.component';
import {ConfigService} from '../../../../shared/services';
import {
  configKindFromPath,
  ConfigDialogData,
  ConfigPath,
  ConfigRelationRole,
  dialogByKind,
  RelatedConfigDescriptor,
  relatedConfigDescriptors,
} from '../../func';
import {RouterExtraService} from '../../services/router-extra.service';
import {AsyncPipe, Location} from '@angular/common';
import {ConfigShortcutHelpersComponent} from '../../components/shortcut/shortcut.component';
import {CrawlExecutionStatusPipe, JobExecutionStatusPipe} from '../../pipe';
import {SeedDialogComponent} from '../../components/seed/seed-dialog/seed-dialog.component';
import {ConfigQuery} from '../../../../shared/func';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {AppConfig} from '../../../../app.config';
import {ResolvedLabelLink, resolveLabelLink} from '../../func';
import {configKindIcon} from '../../func/config-kind-icon';
import {DetailHeaderComponent} from '../../../../shared/components';


export interface ConfigOptions {
  rotationPolicies?: RotationPolicy[];
  subCollectionTypes?: SubCollectionType[];
  crawlConfigs?: ConfigObject[];
  crawlScheduleConfigs?: ConfigObject[];
  browserConfigs?: ConfigObject[];
  collections?: ConfigObject[];
  politenessConfigs?: ConfigObject[];
  browserScripts?: ConfigObject[];
  scopeScripts?: ConfigObject[];
  browserScriptTypes?: BrowserScriptType[];
  robotsPolicies?: RobotsPolicy[];
  crawlJobs?: ConfigObject[];
  roles?: Role[];
}

export interface RelatedConfigContext {
  descriptor: RelatedConfigDescriptor;
  configObject: ConfigObject | null;
  unavailable: boolean;
}

export interface RelatedConfigGroup {
  role: ConfigRelationRole;
  label: string;
  contexts: RelatedConfigContext[];
}

type ConfigDialogContext = 'current' | 'related';

@Component({
  selector: 'app-configuration',
  templateUrl: './configuration.component.html',
  styleUrls: ['./configuration.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    BrowserScriptDetailsComponent,
    BrowserConfigDetailsComponent,
    CollectionDetailsComponent,
    ConfigLabelLinksComponent,
    ConfigContextCardComponent,
    CrawlConfigDetailsComponent,
    CrawlExecutionStatusComponent,
    CrawlExecutionStatusPipe,
    CrawlHostGroupConfigDetailsComponent,
    CrawlJobDetailsComponent,
    DetailHeaderComponent,
    EntityDetailsComponent,
    EntitySeedContextComponent,
    JobExecutionStatusPipe,
    JobStatusComponent,
    PolitenessConfigDetailsComponent,
    ScheduleDetailsComponent,
    SeedDetailsComponent,
    ConfigShortcutHelpersComponent,
    RoleMappingDetailsComponent,
    RouterLink,
  ],
  standalone: true
})
export class ConfigurationComponent implements OnDestroy {
  protected authService = inject(AuthService);
  private dataService = inject(ConfigService);
  private snackBarService = inject(SnackBarService);
  private errorHandler = inject(ErrorHandler);
  private router = inject(Router);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private optionsService = inject(OptionsService);
  private optionsResolver = inject(OptionsResolver);
  private controllerApiService = inject(ControllerApiService);
  private routerExtraService = inject(RouterExtraService);
  private location = inject(Location);
  private destroyRef = inject(DestroyRef);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private appConfig = inject(AppConfig);

  readonly Kind = Kind;
  readonly configKindIcon = configKindIcon;
  readonly canReadSeeds = computed(() => this.abilityService.can('read', Kind[Kind.SEED]));
  readonly newConfigTitle = $localize`:@@commonConfigDetailsCardSubtitle:New (unsaved)`;

  private ngUnsubscribe = new Subject<void>();

  private configObject: Subject<ConfigObject>;
  configObject$: Observable<ConfigObject>;
  readonly entitySeedDisabled = signal<boolean | null>(null);
  entitySeedDataSource: ListDataSource<ConfigObject, ConfigQuery>;
  relatedConfigs$: Observable<RelatedConfigContext[]>;
  readonly relatedConfigContexts: Signal<RelatedConfigContext[]>;
  scriptAnnotationContexts$: Observable<ScriptAnnotationContext[]>;
  annotationSuggestions$: Observable<string[]>;

  private readonly reload = signal(0);

  options: ConfigOptions;
  options$: Observable<ConfigOptions>;

  constructor() {
    this.configObject = new Subject();

    this.options$ = this.optionsService.options$.pipe(
      tap(options => {
        this.options = options;
      }),
    );

    const paramMap = toSignal(this.route.paramMap, {requireSync: true});
    const parentParamMap = toSignal(this.route.parent.paramMap, {requireSync: true});
    const configRef = computed(() => {
      this.reload();
      return new ConfigRef({
        kind: configKindFromPath(parentParamMap().get('kind')),
        id: paramMap().get('id'),
      });
    });

    const configObject$: Observable<ConfigObject> = toObservable(configRef).pipe(
      switchMap(configRef =>
        configRef && configRef.id ? this.dataService.get(configRef) : of(null))
    );

    this.configObject$ = merge(this.configObject.asObservable(), configObject$).pipe(
      shareReplay({bufferSize: 1, refCount: true}),
    );
    const entitySeedQuery$ = combineLatest([
      toObservable(configRef),
      toObservable(this.entitySeedDisabled),
      toObservable(this.canReadSeeds),
    ]).pipe(
      map(([ref, disabled, canReadSeeds]) => this.entitySeedQuery(ref, disabled, canReadSeeds)),
    );
    this.entitySeedDataSource = ListDataSource.fromQuery({
      query$: entitySeedQuery$,
      load: (query, range) => query.kind === Kind.SEED ? this.dataService.search(query, range) : EMPTY,
      destroyRef: this.destroyRef,
    });
    this.relatedConfigs$ = combineLatest([this.configObject$, this.options$]).pipe(
      switchMap(([configObject, options]) => {
        const descriptors = relatedConfigDescriptors(configObject, options?.browserScripts)
          .filter(({ref}) => this.authService.canRead(ref.kind));
        if (!descriptors.length) {
          return of([] as RelatedConfigContext[]);
        }
        return forkJoin(descriptors.map(descriptor => this.dataService.get(descriptor.ref).pipe(
          map(related => ({descriptor, configObject: related, unavailable: false} as RelatedConfigContext)),
          catchError(() => of({descriptor, configObject: null, unavailable: true} as RelatedConfigContext)),
        )));
      }),
    );
    this.relatedConfigContexts = toSignal(this.relatedConfigs$, {initialValue: []});

    this.scriptAnnotationContexts$ = combineLatest([this.configObject$, this.options$]).pipe(
      switchMap(([configObject, options]) => this.loadScriptAnnotationContexts(configObject, options)),
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.annotationSuggestions$ = this.scriptAnnotationContexts$.pipe(
      map(contexts => Array.from(new Set(
        contexts.flatMap(context => context.annotations.map(annotation => annotation.key))
      )).sort((a, b) => a.localeCompare(b))),
    );
  }

  get loading$(): Observable<boolean> {
    return this.dataService.loading$;
  }

  crawlJobNameFor(jobId: string, crawlJobs: ConfigObject[] = []): string {
    return crawlJobs.find(crawlJob => crawlJob.id === jobId)?.meta.name?.trim() ?? '';
  }

  configKindLabel(kind: Kind): string {
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
      case Kind.CRAWLHOSTGROUPCONFIG: return $localize`:@@configKindCrawlHostGroup:Crawl host group`;
      case Kind.ROLEMAPPING: return $localize`:@@configKindRoleMapping:User`;
      default: return $localize`:@@configKindConfiguration:Configuration`;
    }
  }

  relatedConfigGroups(contexts: readonly RelatedConfigContext[]): RelatedConfigGroup[] {
    const groups = new Map<ConfigRelationRole, RelatedConfigContext[]>();
    for (const context of contexts) {
      const role = context.descriptor.role;
      groups.set(role, [...(groups.get(role) ?? []), context]);
    }
    return [...groups.entries()].map(([role, groupContexts]) => ({
      role,
      label: this.relationshipLabel(role),
      contexts: groupContexts,
    }));
  }

  relationshipLabel(role: ConfigRelationRole): string {
    switch (role) {
      case 'entity': return $localize`:@@configRelationEntity:Entity`;
      case 'crawl-job': return $localize`:@@configRelationCrawlJobs:Crawl jobs`;
      case 'schedule': return $localize`:@@configRelationSchedule:Schedule`;
      case 'crawl-config': return $localize`:@@configRelationCrawlConfig:Crawl config`;
      case 'scope-script': return $localize`:@@configRelationScopeScript:Scope script`;
      case 'collection': return $localize`:@@configRelationCollection:Collection`;
      case 'browser-config': return $localize`:@@configRelationBrowserConfig:Browser config`;
      case 'politeness-config': return $localize`:@@configRelationPoliteness:Politeness config`;
      case 'browser-script': return $localize`:@@configRelationBrowserScripts:Browser scripts`;
    }
  }

  labelLinksFor(configObject: ConfigObject): ResolvedLabelLink[] {
    const seen = new Set<string>();
    return (configObject?.meta?.labelList ?? [])
      .map(label => resolveLabelLink(this.appConfig.labelLinks, label))
      .filter((link): link is ResolvedLabelLink => {
        if (!link) {
          return false;
        }
        const key = `${link.text}:${link.href}`;
        if (seen.has(key)) {
          return false;
        }
        seen.add(key);
        return true;
      });
  }

  onEntitySeedStatusChange(disabled: boolean | null): void {
    this.entitySeedDisabled.set(disabled);
  }

  onOpenEntitySeed(seed: ConfigObject): void {
    this.router.navigate(['/config', 'seed', seed.id])
      .catch(error => this.errorHandler.handleError(error));
  }

  onEditEntitySeed(seed: ConfigObject): void {
    this.onCreateConfigWithDialog(seed, 'related');
  }

  private entitySeedQuery(
    configRef: ConfigRef,
    disabled: boolean | null,
    canReadSeeds: boolean,
  ): ConfigQuery {
    const isReadableEntity = configRef?.kind === Kind.CRAWLENTITY
      && !!configRef.id
      && canReadSeeds;
    return {
      kind: isReadableEntity ? Kind.SEED : Kind.UNDEFINED,
      entityId: isReadableEntity ? configRef.id : null,
      scheduleId: null,
      crawlConfigId: null,
      collectionId: null,
      browserConfigId: null,
      politenessId: null,
      disabled,
      browserScriptType: null,
      robotsPolicy: null,
      role: null,
      crawlJobIdList: [],
      scriptIdList: [],
      term: null,
      active: '',
      direction: '',
    };
  }

  private loadScriptAnnotationContexts(
    configObject: ConfigObject,
    options: ConfigOptions | null,
  ): Observable<ScriptAnnotationContext[]> {
    const jobs = this.scriptAnnotationJobs(configObject, options);
    if (!jobs.length) {
      return of([]);
    }

    const seedId = configObject.kind === Kind.SEED ? configObject.id : undefined;
    return forkJoin(jobs.map(job => this.dataService.getScriptAnnotations(job.ref.id, seedId).pipe(
      map((annotations: Annotation[]) => ({
        jobRef: job.ref,
        jobName: job.name,
        annotations,
        unavailable: false,
      } as ScriptAnnotationContext)),
      catchError(() => of({
        jobRef: job.ref,
        jobName: job.name,
        annotations: [],
        unavailable: true,
      } as ScriptAnnotationContext)),
    )));
  }

  private scriptAnnotationJobs(
    configObject: ConfigObject,
    options: ConfigOptions | null,
  ): {ref: ConfigRef; name: string}[] {
    let jobs: {ref: ConfigRef; name: string}[] = [];

    switch (configObject?.kind) {
      case Kind.CRAWLJOB:
        if (configObject.id) {
          jobs = [{
            ref: ConfigObject.toConfigRef(configObject),
            name: configObject.meta.name || configObject.id,
          }];
        }
        break;
      case Kind.SEED:
        jobs = (configObject.seed?.jobRefList ?? []).map(ref => ({
          ref,
          name: options?.crawlJobs?.find(job => job.id === ref.id)?.meta.name || ref.id,
        }));
        break;
      case Kind.CRAWLENTITY:
        jobs = (options?.crawlJobs ?? []).map(job => ({
          ref: ConfigObject.toConfigRef(job),
          name: job.meta.name || job.id,
        }));
        break;
    }

    const seen = new Set<string>();
    return jobs.filter(job => {
      if (!job.ref?.id || seen.has(job.ref.id)) {
        return false;
      }
      seen.add(job.ref.id);
      return true;
    });
  }

  ngOnDestroy(): void {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  onCreateSeedFromEntity(entity: ConfigObject) {
    const entityRef = ConfigObject.toConfigRef(entity);
    const configObject = new ConfigObject({kind: Kind.SEED, seed: new Seed({entityRef})});

    this.onCreateConfigWithDialog(configObject, 'related');
  }

  onCreateConfigWithDialog(configObject: ConfigObject, context: ConfigDialogContext = 'current') {
    if (!configObject) {
      return;
    }

    this.optionsResolver.load(configObject.kind).pipe(
      take(1),
      takeUntil(this.ngUnsubscribe),
    ).subscribe({
      next: kindOptions => {
        const options = {...this.options, ...kindOptions};
        this.optionsService.next(kindOptions);
        this.openConfigDialog(configObject, options, context);
      },
      error: error => this.errorHandler.handleError(error),
    });
  }

  private openConfigDialog(configObject: ConfigObject, options: ConfigOptions, context: ConfigDialogContext) {
    const data: ConfigDialogData = {configObject, options};
    const componentType = dialogByKind(configObject.kind);
    const dialogRef = this.dialog.open(componentType, {data});

    if (configObject.kind === Kind.SEED) {
      const move = (dialogRef.componentInstance as SeedDialogComponent).move.subscribe((parcel: Parcel) => {
        this.onMoveSeed(parcel);
        move.unsubscribe();
      });
    }

    this.router.events.pipe(
      filter(event => event instanceof NavigationStart),
      tap(() => this.dialog.closeAll()),
      takeUntil(this.ngUnsubscribe),
    ).subscribe();

    dialogRef.afterClosed().pipe(
      filter(_ => !!_),
      takeUntil(this.ngUnsubscribe),
    ).subscribe((config: ConfigObject) => {
      if (config.id) {
        if (context === 'current') {
          this.onUpdateConfig(config);
        } else {
          this.onUpdateRelatedConfig(config);
        }
      } else if (context === 'current') {
        this.onSaveConfig(config);
      } else {
        this.onSaveRelatedConfig(config);
      }
    });
  }

  private onUpdateRelatedConfig(configObject: ConfigObject): void {
    this.dataService.update(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(() => {
        this.entitySeedDataSource.refreshLoaded();
        this.reload.update(value => value + 1);
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.updated:Updated`);
      });
  }

  private onSaveRelatedConfig(configObject: ConfigObject): void {
    this.dataService.save(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(() => {
        this.entitySeedDataSource.refreshLoaded();
        this.reload.update(value => value + 1);
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.saved:Saved`);
      });
  }

  onEditRelatedConfig(configObject: ConfigObject): void {
    this.onCreateConfigWithDialog(configObject, 'related');
  }

  onCloneRelatedConfig(configObject: ConfigObject): void {
    this.onCreateConfigWithDialog(ConfigObject.clone(configObject), 'related');
  }

  onClone(configObject: ConfigObject) {
    this.onCreateConfigWithDialog(ConfigObject.clone(configObject));
  }

  onSaveConfig(configObject: ConfigObject) {
    this.dataService.save(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(newConfig => {
        this.configObject.next(newConfig);
        this.router.navigate(['/config', ConfigPath[newConfig.kind], newConfig.id])
          .catch(error => this.errorHandler.handleError(error));
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.saved:Saved`);
      });
  }

  onUpdateConfig(configObject: ConfigObject) {
    this.dataService.update(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(newConfig => {
        this.configObject.next(newConfig);
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.updated:Updated`);
      });
  }


  onDeleteConfig(configObject: ConfigObject) {
    this.deleteConfig(configObject, true);
  }

  onDeleteRelatedConfig(configObject: ConfigObject): void {
    this.deleteConfig(configObject, false);
  }

  private deleteConfig(configObject: ConfigObject, navigateAfterDelete: boolean): void {
    const dialogRef = this.dialog.open(DeleteDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {configObject},
    });
    const previousUrl = this.routerExtraService.getPreviousUrl();
    const currentUrl = this.routerExtraService.getCurrentUrl();
    dialogRef.afterClosed()
      .pipe(
        filter(result => !!result),
        switchMap(() => this.dataService.delete(configObject)),
        filter(deleted => !!deleted)
      )
      .subscribe(() => {
        if (!navigateAfterDelete) {
          this.entitySeedDataSource.refreshLoaded();
          this.reload.update(value => value + 1);
        } else if (previousUrl !== currentUrl) {
          this.location.back();
        } else {
          this.router.navigate(['../'], {
            relativeTo: this.route,
          }).catch(error => this.errorHandler.handleError(error));
        }
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.deleted:Deleted`);
      });
  }

  onSaveMultipleSeeds(configObjects: ConfigObject[]) {
    this.dataService.saveMultiple(configObjects)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(saved => {
        this.snackBarService.openSnackBar(saved + $localize`:@snackBarMessage.multipleSaved: configurations saved`);
        this.configObject.next(null);
        this.reload.update(value => value + 1);
      });
  }

  onMoveSeed(parcel: Parcel) {
    (Array.isArray(parcel.seed)
      ? this.dataService.moveMultiple(parcel.seed, parcel.entityRef)
      : this.dataService.move(parcel.seed, parcel.entityRef))
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(moved => {
        if (moved > 0) {
          this.reload.update(value => value + 1);
        }
        this.snackBarService.openSnackBar(moved + $localize`:@snackBarMessage.multipleMoved: configurations moved`);
      });
  }

  onRunCrawl(configObject: ConfigObject) {
    this.openRunCrawlDialog(configObject);
  }

  onRunSeedInCrawlJob(seed: ConfigObject, crawlJob: ConfigObject) {
    this.openRunCrawlDialog(seed, crawlJob.id);
  }

  private openRunCrawlDialog(configObject: ConfigObject, jobRefId?: string) {
    const crawlJobs = this.options.crawlJobs;
    const dialogRef = this.dialog.open(RunCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {configObject, jobRefId, crawlJobs}
    });
    dialogRef.afterClosed()
      .subscribe(result => {
        if (result?.runCrawlRequest) {
          this.controllerApiService.runCrawl(result.runCrawlRequest)
            .subscribe(runCrawlReply => {
              if (configObject.kind === Kind.SEED) {
                this.router.navigate(
                  ['report', 'crawlexecution'],
                  {
                    queryParams: {
                      job_execution_id: runCrawlReply.jobExecutionId,
                      seed_id: configObject.id,
                    }
                  }
                ).catch(error => this.errorHandler.handleError(error));
              } else {
                this.router.navigate(
                  ['report', 'jobexecution', runCrawlReply.jobExecutionId],
                  {queryParams: {watch: true}}
                ).catch(error => this.errorHandler.handleError(error));
              }
            });
        }
      });
  }
}
